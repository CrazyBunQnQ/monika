#!/usr/bin/env python3
"""
Quick structure verification script for Task 1 setup.
Verifies that all required files exist without importing dependencies.
"""
import sys
import os

print("=" * 60)
print("Monika Backend - Task 1 Structure Verification")
print("=" * 60)

# Define required files
required_files = {
    "Backend Core": [
        "backend/__init__.py",
        "backend/core/__init__.py",
        "backend/core/config.py",
        "backend/core/database.py",
    ],
    "Configuration": [
        "backend/requirements.txt",
        ".env.example",
        "docker-compose.yml",
        ".gitignore",
    ],
    "Alembic": [
        "backend/alembic.ini",
        "backend/alembic/env.py",
        "backend/alembic/script.py.mako",
        "backend/alembic/versions/.gitkeep",
    ],
    "Tests": [
        "backend/tests/__init__.py",
        "backend/tests/test_config.py",
        "backend/tests/test_database.py",
    ],
    "Documentation": [
        "backend/README.md",
    ]
}

base_path = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
all_passed = True

for category, files in required_files.items():
    print(f"\n{category}:")
    for file_path in files:
        full_path = os.path.join(base_path, file_path)
        if os.path.exists(full_path):
            print(f"   ✓ {file_path}")
        else:
            print(f"   ✗ {file_path} - MISSING")
            all_passed = False

print("\n" + "=" * 60)
if all_passed:
    print("✓ All required files are present!")
else:
    print("✗ Some files are missing!")
    sys.exit(1)
print("=" * 60)
print("\nTask 1 Implementation Summary:")
print("- Database configuration: backend/core/config.py")
print("- Database session management: backend/core/database.py")
print("- Docker services: docker-compose.yml")
print("- Alembic migrations: backend/alembic/")
print("- Test files: backend/tests/")
print("\nNext steps:")
print("1. Install dependencies: pip install -r backend/requirements.txt")
print("2. Start Docker: docker-compose up -d")
print("3. Run tests: pytest backend/tests/")
print("=" * 60)
