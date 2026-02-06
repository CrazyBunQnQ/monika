import pytest
import os
import sys

# Add backend to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

def test_migration_file_exists():
    """Test that the migration file exists."""
    from pathlib import Path
    migration_dir = Path(__file__).parent.parent / "alembic" / "versions"
    migration_files = list(migration_dir.glob("*.py"))
    # Filter out __pycache__ and .gitkeep
    migration_files = [f for f in migration_files if f.name != "__init__.py" and f.name != ".gitkeep"]
    assert len(migration_files) > 0, "No migration files found"

    # Check for users table migration
    users_migration = any("create_users" in f.name.lower() or "users" in f.name.lower() for f in migration_files)
    assert users_migration, "No users table migration found"


def test_migration_imports():
    """Test that the migration file can be imported."""
    import importlib.util
    from pathlib import Path

    migration_dir = Path(__file__).parent.parent / "alembic" / "versions"
    migration_files = list(migration_dir.glob("*create_users*.py"))

    assert len(migration_files) > 0, "No users migration file found"

    # Import the migration module
    spec = importlib.util.spec_from_file_location("migration", migration_files[0])
    assert spec is not None, "Could not load migration spec"

    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)

    # Verify migration has required functions
    assert hasattr(module, 'upgrade'), "Migration missing upgrade function"
    assert hasattr(module, 'downgrade'), "Migration missing downgrade function"
    assert hasattr(module, 'revision'), "Migration missing revision id"
