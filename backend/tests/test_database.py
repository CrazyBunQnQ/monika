"""
Test database configuration.

This test validates that the database module can be imported
and has the expected components.
"""


def test_database_module_imports():
    """Test that database module can be imported."""
    from core import database

    assert hasattr(database, "engine")
    assert hasattr(database, "SessionLocal")
    assert hasattr(database, "Base")
    assert hasattr(database, "get_db")


def test_base_is_declarative():
    """Test that Base is a declarative base."""
    from core.database import Base

    # Check that Base has the metadata attribute
    assert hasattr(Base, "metadata")


def test_get_db_is_generator():
    """Test that get_db is a generator function."""
    from core.database import get_db
    import inspect

    assert inspect.isgeneratorfunction(get_db)
