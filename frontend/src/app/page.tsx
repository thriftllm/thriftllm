"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth";
import { getSetupStatus } from "@/lib/api";

export default function Home() {
  const router = useRouter();
  const { user, loading } = useAuth();

  useEffect(() => {
    if (loading) return;

    async function check() {
      try {
        const status = await getSetupStatus();
        if (!status.is_complete) {
          router.replace("/setup");
          return;
        }
        if (user) {
          router.replace("/overview");
        } else {
          router.replace("/login");
        }
      } catch {
        router.replace("/login");
      }
    }
    check();
  }, [loading, user, router]);

  return (
    <div className="flex h-screen items-center justify-center">
      <div className="flex flex-col items-center gap-4">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        <p className="text-sm text-muted-foreground">Loading ThriftLLM...</p>
      </div>
    </div>
  );
}
