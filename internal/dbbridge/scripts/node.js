"use strict";
const readline = require("readline");

const conns = new Map();
const rl = readline.createInterface({ input: process.stdin });
rl.on("line", (line) => {
  let req;
  try {
    req = JSON.parse(line);
  } catch (e) {
    return respond({ id: 0, status: "error", error: "invalid json" });
  }
  handle(req).catch((e) => respond({ id: req.id, status: "error", error: String(e.message || e) }));
});

function respond(resp) {
  process.stdout.write(JSON.stringify(resp) + "\n");
}

async function handle(req) {
  const actions = { ping: doPing, open: doOpen, query: doQuery, schema: doSchema, close: doClose };
  const fn = actions[req.action];
  if (!fn) return respond({ id: req.id, status: "error", error: "unknown action: " + req.action });
  try {
    const result = await fn(req);
    respond({ id: req.id, status: "ok", ...result });
  } catch (e) {
    respond({ id: req.id, status: "error", error: String(e.message || e) });
  }
}

function doPing() {
  return {};
}

async function doOpen(req) {
  const { driver, dsn, conn } = req;
  let client;
  switch (driver) {
    case "postgres": {
      const { Client } = require("pg");
      client = new Client(dsn);
      await client.connect();
      break;
    }
    case "mysql": {
      const mysql = require("mysql2/promise");
      client = await mysql.createConnection(dsn);
      break;
    }
    case "sqlite": {
      const Database = require("better-sqlite3");
      client = new Database(dsn);
      break;
    }
    case "redis": {
      const Redis = require("ioredis");
      client = new Redis(dsn);
      break;
    }
    case "mongo": {
      const { MongoClient } = require("mongodb");
      client = new MongoClient(dsn);
      await client.connect();
      break;
    }
    default:
      throw new Error("unsupported driver: " + driver);
  }
  conns.set(conn, { driver, client });
  return { conn };
}

function getConn(req) {
  const entry = conns.get(req.conn);
  if (!entry) throw new Error("connection not found: " + req.conn);
  return entry;
}

const SQL_READONLY = /^(SELECT|SHOW|DESCRIBE|EXPLAIN)\s/i;
function isWithSelect(q) {
  const upper = q.toUpperCase();
  if (!upper.startsWith("WITH")) return false;
  const forbidden = /\b(INSERT|UPDATE|DELETE)\b/i;
  const selectRe = /\bSELECT\b/i;
  const selIdx = upper.search(selectRe);
  const forbidIdx = upper.search(forbidden);
  if (selIdx === -1) return false;
  if (forbidIdx === -1) return true;
  return selIdx < forbidIdx;
}

const REDIS_READONLY = /^(GET|MGET|KEYS|TYPE|SCAN|HGET|HGETALL|LRANGE|SMEMBERS|ZCARD|ZSCORE|ZRANGE|SCARD|SISMEMBER|EXISTS|TTL|STRLEN)\b/i;

async function doQuery(req) {
  const { driver, client } = getConn(req);
  const q = req.query;

  if (driver === "postgres" || driver === "mysql" || driver === "sqlite") {
    if (!SQL_READONLY.test(q) && !isWithSelect(q)) throw new Error("only read-only queries allowed");
    if (driver === "sqlite") {
      const stmt = client.prepare(q);
      const rows = stmt.all();
      const columns = stmt.columns().map((c) => c.name);
      return { columns, rows: rows.map((r) => columns.map((c) => r[c])), tag: "SELECT" };
    }
    const result = driver === "postgres" ? await client.query(q) : await client.execute(q);
    return { columns: result.fields ? result.fields.map((f) => f.name) : (result.meta || []).map((f) => f.name), rows: result.rows.map((r) => (result.fields || result.meta || []).map((f) => r[f.name])), tag: "SELECT" };
  }

  if (driver === "redis") {
    const parts = q.trim().split(/\s+/);
    if (!REDIS_READONLY.test(parts[0])) throw new Error("only read-only commands allowed");
    const val = await client.call(parts[0], ...parts.slice(1));
    return { columns: ["result"], rows: [[typeof val === "object" && val !== null ? JSON.stringify(val) : val]], tag: "REDIS" };
  }

  if (driver === "mongo") {
    const filter = req.filter ? JSON.parse(req.filter) : {};
    const db = client.db();
    const collection = db.collection(q);
    const docs = await collection.find(filter).limit(100).toArray();
    if (docs.length === 0) return { columns: [], rows: [], tag: "FIND" };
    const columns = Object.keys(docs[0]);
    return { columns, rows: docs.map((d) => columns.map((c) => d[c])), tag: "FIND" };
  }

  throw new Error("unsupported driver for query: " + driver);
}

async function doSchema(req) {
  const { driver, client } = getConn(req);

  if (driver === "postgres") {
    const tables = await client.query("SELECT table_name FROM information_schema.tables WHERE table_schema='public'");
    const result = [];
    for (const t of tables.rows) {
      const cols = await client.query("SELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_name=$1 ORDER BY ordinal_position", [t.table_name]);
      const pkRows = await client.query("SELECT a.attname FROM pg_index i JOIN pg_attribute a ON a.attrelid=i.indrelid AND a.attnum=ANY(i.indkey) WHERE i.indrelid=$1::regclass AND i.indisprimary", [t.table_name]);
      const pks = new Set(pkRows.rows.map((r) => r.attname));
      result.push({ name: t.table_name, columns: cols.rows.map((c) => ({ name: c.column_name, type: c.data_type, nullable: c.is_nullable === "YES", pk: pks.has(c.column_name) })) });
    }
    return { tables: result };
  }

  if (driver === "mysql") {
    const [tableRows] = await client.execute("SHOW TABLES");
    const result = [];
    for (const t of tableRows) {
      const tname = Object.values(t)[0];
      const [colRows] = await client.execute("DESCRIBE " + tname);
      result.push({ name: tname, columns: colRows.map((c) => ({ name: c.Field, type: c.Type, nullable: c.Null === "YES", pk: c.Key === "PRI" })) });
    }
    return { tables: result };
  }

  if (driver === "sqlite") {
    const tables = client.prepare("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").all();
    const result = [];
    for (const t of tables) {
      const cols = client.pragma("table_info(" + t.name + ")");
      result.push({ name: t.name, columns: cols.map((c) => ({ name: c.name, type: c.type, nullable: c.notnull === 0, pk: c.pk > 0 })) });
    }
    return { tables: result };
  }

  if (driver === "redis") {
    const info = await client.info("keyspace");
    const keys = new Set();
    let stream = client.scanStream({ count: 100 });
    await new Promise((resolve, reject) => {
      stream.on("data", (r) => r.forEach((k) => keys.add(k)));
      stream.on("end", resolve);
      stream.on("error", reject);
    });
    const types = new Map();
    for (const k of [...keys].slice(0, 200)) types.set(k, await client.type(k));
    const patterns = new Map();
    for (const [k, v] of types) {
      const pat = k.replace(/[0-9a-f]{8,}(-[0-9a-f]{4,})+/gi, "*").replace(/\d+/g, "*");
      patterns.set(pat, (patterns.get(pat) || 0) + 1);
    }
    const tables = [...patterns.entries()].map(([name, count]) => ({ name, columns: [{ name: "pattern", type: "string", nullable: false, pk: false }, { name: "type", type: "string", nullable: false, pk: false }, { name: "count", type: "integer", nullable: false, pk: false }] }));
    return { tables };
  }

  if (driver === "mongo") {
    const db = client.db();
    const collections = await db.listCollections().toArray();
    const result = [];
    for (const c of collections) {
      const docs = await db.collection(c.name).find().limit(5).toArray();
      if (docs.length === 0) { result.push({ name: c.name, columns: [] }); continue; }
      const typeMap = new Map();
      for (const d of docs) for (const [k, v] of Object.entries(d)) {
        if (!typeMap.has(k)) typeMap.set(k, typeof v);
        else if (typeMap.get(k) !== typeof v) typeMap.set(k, "mixed");
      }
      result.push({ name: c.name, columns: [...typeMap.entries()].map(([name, type]) => ({ name, type, nullable: true, pk: name === "_id" })) });
    }
    return { tables: result };
  }

  throw new Error("unsupported driver for schema: " + driver);
}

async function doClose(req) {
  const entry = conns.get(req.conn);
  if (!entry) return {};
  const { driver, client } = entry;
  conns.delete(req.conn);
  if (driver === "postgres") await client.end();
  else if (driver === "mysql") await client.end();
  else if (driver === "sqlite") client.close();
  else if (driver === "redis") client.disconnect();
  else if (driver === "mongo") await client.close();
  return {};
}
