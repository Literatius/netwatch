from pydantic import BaseModel


class HttpDetails(BaseModel):
    status_code: int

class DNSDetails(BaseModel):
    records: list[str]

class TLSDetails(BaseModel):
    issuer : str
    not_after: str
    days_remaining: int
    dns_names: list[str]

class CheckResult(BaseModel):
    type: str
    target: str
    status: str
    latency: float
    http: HttpDetails | None = None
    dns: DNSDetails | None = None
    tls: TLSDetails | None = None
