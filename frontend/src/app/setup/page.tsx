"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth";
import { setupAdmin } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Zap, ArrowRight } from "lucide-react";

export default function SetupPage() {
  const router = useRouter();
  const { refreshUser } = useAuth();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      await setupAdmin({ name, email, password });
      await refreshUser();
      router.push("/overview");
    } catch (err: any) {
      setError(err.message || "Setup failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-background via-background to-muted/50 p-4">
      <div className="w-full max-w-md animate-fade-in">
        <div className="text-center mb-8">
          <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-primary text-primary-foreground shadow-lg shadow-primary/25">
            <Zap className="h-7 w-7" />
          </div>
          <h1 className="text-2xl font-bold tracking-tight">ThriftLLM</h1>
          <p className="text-sm text-muted-foreground mt-1">Self-hosted LLM proxy</p>
        </div>
        <Card className="border-border/50 shadow-xl shadow-black/5">
          <CardHeader className="text-center pb-2">
            <CardTitle className="text-lg">Get Started</CardTitle>
            <CardDescription className="text-xs">Create your admin account to begin</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              {error && (
                <div className="rounded-lg bg-destructive/10 border border-destructive/20 px-3 py-2 text-xs text-destructive">{error}</div>
              )}
              <div className="space-y-1.5">
                <Label htmlFor="name" className="text-xs font-medium">Name</Label>
                <Input id="name" className="h-9" value={name} onChange={(e) => setName(e.target.value)} placeholder="Admin" required />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="email" className="text-xs font-medium">Email</Label>
                <Input id="email" className="h-9" type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="admin@example.com" required />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="password" className="text-xs font-medium">Password</Label>
                <Input id="password" className="h-9" type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Min 8 characters" required minLength={8} />
              </div>
              <Button type="submit" className="w-full h-9" disabled={loading}>
                {loading ? "Creating..." : (<>Create Admin Account <ArrowRight className="h-3.5 w-3.5 ml-1.5" /></>)}
              </Button>
            </form>
          </CardContent>
        </Card>
        <div className="mt-6 flex items-center justify-center gap-4 text-[11px] text-muted-foreground/60">
          <span>Multi-provider</span>
          <span className="h-1 w-1 rounded-full bg-muted-foreground/30" />
          <span>Semantic caching</span>
          <span className="h-1 w-1 rounded-full bg-muted-foreground/30" />
          <span>Smart routing</span>
        </div>
      </div>
    </div>
  );
}
