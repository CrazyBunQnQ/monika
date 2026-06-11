import sys
import json
import re
from urllib.parse import urlparse, unquote

conns = {}

def _quote_id(name):
    return '"' + name.replace('"', '""') + '"'

def _quote_mysql_id(name):
    return '`' + name.replace('`', '``') + '`'

SQL_READONLY = re.compile(r"^(SELECT|SHOW|DESCRIBE|EXPLAIN)\s", re.IGNORECASE)
SQL_FORBIDDEN = re.compile(r"INTO\s+OUTFILE|INTO\s+DUMPFILE|FOR\s+UPDATE|FOR\s+SHARE|INTO\s+@", re.IGNORECASE)
REDIS_READONLY = re.compile(
    r"^(GET|MGET|TYPE|SCAN|HGET|HGETALL|LRANGE|SMEMBERS|ZCARD|ZSCORE|ZRANGE|SCARD|SISMEMBER|EXISTS|TTL|STRLEN)\b",
    re.IGNORECASE,
)


def respond(resp):
    sys.stdout.write(json.dumps(resp) + "\n")
    sys.stdout.flush()


def is_readonly_cte(q):
    upper = q.upper()
    start = 5
    if upper.startswith("WITH RECURSIVE "):
        start = 16
    depth = 0
    in_str = False
    i = start
    while i < len(upper):
        ch = upper[i]
        if in_str:
            if ch == "'":
                in_str = False
            i += 1
            continue
        if ch == "'":
            in_str = True
            i += 1
            continue
        if ch == '(':
            depth += 1
        elif ch == ')':
            depth -= 1
            if depth == 0:
                rest = upper[i + 1:].strip()
                if re.match(r"^(SELECT|SHOW|DESCRIBE|EXPLAIN)\s", rest):
                    return True
                if re.match(r"^(INSERT|UPDATE|DELETE|DROP|CREATE|ALTER|TRUNCATE)", rest):
                    return False
        i += 1
    return False


def get_conn(req):
    name = req.get("conn", "")
    if name not in conns:
        raise Exception("connection not found: " + name)
    return conns[name]


def do_ping(_req):
    return {}


def do_open(req):
    driver = req.get("driver", "")
    dsn = req.get("dsn", "")
    conn_name = req.get("conn", "")

    if driver == "postgres":
        import psycopg2

        client = psycopg2.connect(dsn)
        client.set_session(autocommit=True)
    elif driver == "mysql":
        import pymysql

        if "@tcp(" in dsn:
            at_idx = dsn.index("@tcp(")
            user_pass = dsn[:at_idx]
            rest = dsn[at_idx + 5:]
            if ":" in user_pass:
                user, password = user_pass.split(":", 1)
            else:
                user, password = user_pass, ""
            password = unquote(password)
            paren_close = rest.index(")")
            host_port = rest[:paren_close]
            after = rest[paren_close + 1 :]
            if ":" in host_port:
                host, port = host_port.split(":", 1)
                port = int(port)
            else:
                host, port = host_port, 3306
            dbname = after.lstrip("/") if after.startswith("/") else ""
            client = pymysql.connect(
                host=host,
                port=port,
                user=user,
                password=password,
                database=dbname,
                autocommit=True,
            )
        else:
            result = urlparse(dsn)
            client = pymysql.connect(
                host=result.hostname or "localhost",
                port=result.port or 3306,
                user=result.username or "",
                password=unquote(result.password or ""),
                database=result.path.lstrip("/") if result.path else "",
                autocommit=True,
            )
    elif driver == "sqlite":
        import sqlite3

        client = sqlite3.connect(dsn)
        client.row_factory = sqlite3.Row
    elif driver == "redis":
        import redis

        client = redis.from_url(dsn)
    elif driver == "mongo":
        import pymongo

        client = pymongo.MongoClient(dsn)
    else:
        raise Exception("unsupported driver: " + driver)

    conns[conn_name] = {"driver": driver, "client": client}
    return {"conn": conn_name}


def _validate_mongo_filter(obj):
    forbidden = {"$where", "$function", "$accumulator"}
    if isinstance(obj, dict):
        for k, v in obj.items():
            if k in forbidden:
                raise Exception(f"Forbidden MongoDB operator: {k}")
            _validate_mongo_filter(v)
    elif isinstance(obj, list):
        for item in obj:
            _validate_mongo_filter(item)


def do_query(req):
    entry = get_conn(req)
    driver = entry["driver"]
    client = entry["client"]
    q = req.get("query", "")

    if driver in ("postgres", "mysql", "sqlite"):
        if ";" in q:
            raise Exception("multi-statement queries are not allowed")
        if SQL_FORBIDDEN.search(q):
            raise Exception("write operation detected in query")
        if not SQL_READONLY.match(q) and not is_readonly_cte(q):
            raise Exception("only read-only queries allowed")

        if driver == "sqlite":
            cur = client.execute(q)
            columns = [desc[0] for desc in cur.description] if cur.description else []
            rows = [list(row) for row in cur.fetchall()]
            return {"columns": columns, "rows": rows, "tag": "SELECT"}

        cur = client.cursor()
        cur.execute(q)
        columns = [desc[0] for desc in cur.description] if cur.description else []
        rows = [list(row) for row in cur.fetchall()]
        cur.close()
        return {"columns": columns, "rows": rows, "tag": "SELECT"}

    if driver == "redis":
        parts = q.strip().split()
        if not parts or not REDIS_READONLY.match(parts[0]):
            raise Exception("only read-only commands allowed")
        val = client.execute_command(*parts)
        if isinstance(val, (bytes,)):
            val = val.decode()
        display = json.dumps(val) if isinstance(val, (list, dict)) else val
        return {"columns": ["result"], "rows": [[display]], "tag": "REDIS"}

    if driver == "mongo":
        import pymongo

        try:
            query_obj = json.loads(q)
        except (json.JSONDecodeError, TypeError):
            query_obj = {"collection": q, "filter": {}}
        collection_name = query_obj.get("collection", "unknown")
        filt = query_obj.get("filter", {})
        limit = query_obj.get("limit", 100)
        _validate_mongo_filter(filt)
        db = client.get_default_database() or client.get_database()
        docs = list(db[collection_name].find(filt).limit(limit))
        if not docs:
            return {"columns": [], "rows": [], "tag": "FIND"}
        columns = list(docs[0].keys())
        rows = [[doc.get(c) for c in columns] for doc in docs]
        return {"columns": columns, "rows": rows, "tag": "FIND"}

    raise Exception("unsupported driver for query: " + driver)


def do_schema(req):
    entry = get_conn(req)
    driver = entry["driver"]
    client = entry["client"]

    if driver == "postgres":
        cur = client.cursor()
        cur.execute(
            "SELECT table_name FROM information_schema.tables WHERE table_schema='public'"
        )
        tables = []
        for (tname,) in cur.fetchall():
            cur.execute(
                "SELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_name=%s ORDER BY ordinal_position",
                (tname,),
            )
            cols = cur.fetchall()
            cur.execute(
                "SELECT a.attname FROM pg_index i JOIN pg_attribute a ON a.attrelid=i.indrelid AND a.attnum=ANY(i.indkey) WHERE i.indrelid=%s::regclass AND i.indisprimary",
                (tname,),
            )
            pks = {r[0] for r in cur.fetchall()}
            tables.append(
                {
                    "name": tname,
                    "columns": [
                        {
                            "name": c[0],
                            "type": c[1],
                            "nullable": c[2] == "YES",
                            "pk": c[0] in pks,
                        }
                        for c in cols
                    ],
                }
            )
        cur.close()
        return {"tables": tables}

    if driver == "mysql":
        cur = client.cursor()
        cur.execute("SHOW TABLES")
        tables = []
        for (tname,) in cur.fetchall():
            cur.execute("DESCRIBE " + _quote_mysql_id(tname))
            cols = cur.fetchall()
            tables.append(
                {
                    "name": tname,
                    "columns": [
                        {
                            "name": c[0],
                            "type": c[1],
                            "nullable": c[2] == "YES",
                            "pk": c[3] == "PRI",
                        }
                        for c in cols
                    ],
                }
            )
        cur.close()
        return {"tables": tables}

    if driver == "sqlite":
        cur = client.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
        )
        tables = []
        for (tname,) in cur.fetchall():
            cols = client.execute("PRAGMA table_info(" + _quote_id(tname) + ")").fetchall()
            tables.append(
                {
                    "name": tname,
                    "columns": [
                        {
                            "name": c[1],
                            "type": c[2],
                            "nullable": c[3] == 0,
                            "pk": c[5] > 0,
                        }
                        for c in cols
                    ],
                }
            )
        return {"tables": tables}

    if driver == "redis":
        keys = set()
        cursor = 0
        while len(keys) < 200:
            cursor, batch = client.scan(cursor, count=100)
            for k in batch:
                keys.add(k)
                if len(keys) >= 200:
                    break
            if cursor == 0:
                break
        sample = list(keys)[:200]
        patterns = {}
        for k in sample:
            kstr = k.decode() if isinstance(k, bytes) else k
            ktype = client.type(k)
            ktype = ktype.decode() if isinstance(ktype, bytes) else ktype
            pat = re.sub(r"[0-9a-f]{8,}(-[0-9a-f]{4,})+", "*", kstr)
            pat = re.sub(r"\d+", "*", pat)
            patterns[pat] = patterns.get(pat, 0) + 1
        tables = [
            {
                "name": name,
                "columns": [
                    {"name": "pattern", "type": "string", "nullable": False, "pk": False},
                    {"name": "type", "type": "string", "nullable": False, "pk": False},
                    {"name": "count", "type": "integer", "nullable": False, "pk": False},
                ],
            }
            for name in patterns
        ]
        return {"tables": tables}

    if driver == "mongo":
        db = client.get_default_database() or client.get_database()
        tables = []
        for cinfo in db.list_collections():
            docs = list(db[cinfo["name"]].find().limit(5))
            if not docs:
                tables.append({"name": cinfo["name"], "columns": []})
                continue
            type_map = {}
            for d in docs:
                for k, v in d.items():
                    if k not in type_map:
                        type_map[k] = type(v).__name__
                    elif type_map[k] != type(v).__name__:
                        type_map[k] = "mixed"
            tables.append(
                {
                    "name": cinfo["name"],
                    "columns": [
                        {
                            "name": name,
                            "type": typ,
                            "nullable": True,
                            "pk": name == "_id",
                        }
                        for name, typ in type_map.items()
                    ],
                }
            )
        return {"tables": tables}

    raise Exception("unsupported driver for schema: " + driver)


def do_close(req):
    name = req.get("conn", "")
    if name not in conns:
        return {}
    entry = conns.pop(name)
    driver = entry["driver"]
    client = entry["client"]
    try:
        if driver == "postgres":
            client.close()
        elif driver == "mysql":
            client.close()
        elif driver == "sqlite":
            client.close()
        elif driver == "redis":
            client.close()
        elif driver == "mongo":
            client.close()
    except Exception:
        pass
    return {}


handlers = {
    "ping": do_ping,
    "open": do_open,
    "query": do_query,
    "schema": do_schema,
    "close": do_close,
}

for line in sys.stdin:
    try:
        req = json.loads(line)
    except Exception:
        respond({"id": 0, "status": "error", "error": "invalid json"})
        continue

    rid = req.get("id", 0)
    action = req.get("action", "")
    fn = handlers.get(action)
    if fn is None:
        respond({"id": rid, "status": "error", "error": "unknown action: " + action})
        continue
    try:
        result = fn(req)
        respond({"id": rid, "status": "ok", **result})
    except ImportError as e:
        respond({"id": rid, "status": "error", "error": "driver not installed: " + str(e)})
    except Exception as e:
        respond({"id": rid, "status": "error", "error": str(e)})
