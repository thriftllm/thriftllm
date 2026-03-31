import { NextRequest, NextResponse } from "next/server";

const BACKEND_URL = process.env.BACKEND_URL || "http://localhost:8080";

async function proxyHandler(
  req: NextRequest,
  { params }: { params: Promise<{ path: string[] }> }
) {
  const { path } = await params;
  const backendPath = "/api/" + path.join("/");

  // Build target URL preserving query string
  const url = new URL(backendPath, BACKEND_URL);
  req.nextUrl.searchParams.forEach((value, key) => {
    url.searchParams.set(key, value);
  });

  // Forward auth cookie as Authorization header to backend
  const token = req.cookies.get("thrift_token")?.value;
  const headers: Record<string, string> = {};

  const contentType = req.headers.get("content-type");
  if (contentType) {
    headers["Content-Type"] = contentType;
  }

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  // Forward request body for mutating methods
  let body: string | undefined;
  if (req.method !== "GET" && req.method !== "HEAD") {
    body = await req.text();
  }

  const backendRes = await fetch(url.toString(), {
    method: req.method,
    headers,
    body,
  });

  const data = await backendRes.text();

  // Build response
  const res = new NextResponse(data, {
    status: backendRes.status,
  });

  // Forward content type
  const resContentType = backendRes.headers.get("content-type");
  if (resContentType) {
    res.headers.set("Content-Type", resContentType);
  }

  // Forward Set-Cookie from backend (login/setup/logout set HTTP-only cookies)
  const setCookies = backendRes.headers.getSetCookie();
  for (const cookie of setCookies) {
    res.headers.append("Set-Cookie", cookie);
  }

  return res;
}

export const GET = proxyHandler;
export const POST = proxyHandler;
export const PUT = proxyHandler;
export const PATCH = proxyHandler;
export const DELETE = proxyHandler;
