from fastapi import APIRouter

from api.schemas import CheckResult
from DB.queries import save_check


router = APIRouter()


@router.post("/checks")
def receive_check(check: CheckResult):
    save_check(
        check.type,
        check.target,
        check.status,
        check.latency,
        check.http.status_code if check.http is not None else None,
        check.dns.records if check.dns is not None else None,
        check.tls.model_dump() if check.tls is not None else None
    )

    return {"received": True}
