export const dynamic = "force-dynamic";

import { NextRequest, NextResponse } from "next/server";

const SERVICE = process.env.NEXT_PUBLIC_CLOAK_SERVICE_URL ?? "http://localhost:4000";

export async function GET(req: NextRequest) {
  const keyId = req.nextUrl.searchParams.get("keyId");
  const fmt = req.nextUrl.searchParams.get("fmt") ?? "csv";

  if (!keyId) {
    return NextResponse.json({ error: "keyId required" }, { status: 400 });
  }

  const upstream = await fetch(
    `${SERVICE}/audit/export?keyId=${encodeURIComponent(keyId)}&fmt=${fmt}`
  );

  if (!upstream.ok) {
    return NextResponse.json({ error: "Export failed" }, { status: upstream.status });
  }

  const contentType = upstream.headers.get("content-type") ?? "text/csv";
  const disposition = upstream.headers.get("content-disposition") ?? "";
  const body = await upstream.arrayBuffer();

  return new NextResponse(body, {
    status: 200,
    headers: {
      "Content-Type": contentType,
      "Content-Disposition": disposition,
    },
  });
}
