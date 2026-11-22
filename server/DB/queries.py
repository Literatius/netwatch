import sqlite3
import json
from typing import Any

from logger import logger

def start_db() -> None:
    with sqlite3.connect("monitor.db") as conn:
        cursor = conn.cursor()

        cursor.execute ("""
            CREATE TABLE IF NOT EXISTS checks (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                type TEXT NOT NULL,
                target TEXT NOT NULL,
                status TEXT NOT NULL,
                latency REAL NOT NULL,
                status_code INTEGER NULL,
                dns_records TEXT NULL,
                tls_details TEXT NULL,
                checked_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
            )
            """)

def save_check(
    type:str,
    target: str,
    status: str,
    latency: float,
    status_code: int | None = None,
    dns_records: list[str] | None = None,
    tls: dict [str, Any] | None = None
    ) -> None:
    with sqlite3.connect("monitor.db") as conn:
        cursor = conn.cursor()

        records = json.dumps(dns_records) if dns_records is not None else None
        tls_details = json.dumps(tls) if tls is not None else None

        cursor.execute("""
            INSERT INTO checks (type, target, status, latency, status_code, dns_records, tls_details)
            VALUES (?, ?, ?, ?, ?, ?, ?)
            """,
            (type, target, status, latency, status_code, records, tls_details)
        )

        current_id = cursor.lastrowid

        if type == "dns" and status == "UP" and dns_records is not None:
            previous_id = get_dns_id(conn, current_id, target)

            if previous_id is not None:
                added, removed = compare_dns(conn, previous_id, current_id)


                if added or removed:
                    logger.info(
                        "DNS changed: %s - Added: %s - Removed: %s",
                        target,
                        added,
                        removed
                    )

def get_dns_id(conn: sqlite3.Connection, current_id: int, target: str) -> int | None:
    cursor = conn.cursor()

    previous_query = cursor.execute("""
        SELECT id
        FROM checks
        WHERE type = "dns"
        AND target = ?
        AND status = "UP"
        AND id < ?
        ORDER by id DESC LIMIT 1
        """,
        (target, current_id)).fetchone()

    if previous_query is None:
        return None

    previous_id = previous_query[0]

    return previous_id

def compare_dns(conn: sqlite3.Connection, previous_id: int, current_id: int) -> tuple[list[str], list[str]]:
    query = """
        SELECT DISTINCT a.value
        FROM json_each(
            (SELECT dns_records FROM checks WHERE id = ?)
        ) as a

        LEFT JOIN
        json_each(
            (SELECT dns_records FROM checks where id = ?)
        ) as b

        ON a.value = b.value
        WHERE b.value IS NULL
    """

    added = [row[0] for row in conn.execute(query, (current_id, previous_id))]
    removed = [row[0] for row in conn.execute(query, (previous_id, current_id))]

    return added, removed
