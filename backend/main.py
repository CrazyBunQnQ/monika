"""Main FastAPI application entry point."""
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from backend.api import auth, characters
from backend.core.database import Base, engine


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan handler - creates database tables on startup."""
    # Create tables on startup
    Base.metadata.create_all(bind=engine)
    yield
    # Cleanup if needed


app = FastAPI(
    title="Monika CoC TRPG Platform",
    version="1.0.0",
    description="A Call of Cthulhu TRPG AI Game Master Platform",
    lifespan=lifespan,
)

# CORS middleware configuration
app.add_middleware(
    CORSMiddleware,
    allow_origins=[
        "http://localhost:5173",  # Vite dev server
        "http://localhost:3000",  # Common React dev server
    ],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include routers
app.include_router(auth.router)
app.include_router(characters.router)


@app.get("/")
def root():
    """Root endpoint - returns API information."""
    return {
        "message": "Monika CoC TRPG Platform API",
        "version": "1.0.0",
        "docs": "/docs",
    }


@app.get("/health")
def health():
    """Health check endpoint."""
    return {"status": "healthy"}
