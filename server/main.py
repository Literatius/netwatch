from contextlib import asynccontextmanager

import uvicorn
from fastapi import FastAPI

from DB.queries import start_db
from api.routes import router
from logger import logger


@asynccontextmanager
async def lifespan(app: FastAPI):
    start_db()
    logger.info("Server started")
    yield
    logger.info("Server stopped")


app = FastAPI(lifespan=lifespan)
app.include_router(router)


if __name__ == "__main__":
    uvicorn.run(
        "main:app",
        host="127.0.0.1",
        port=8000,
        reload=True
    )
