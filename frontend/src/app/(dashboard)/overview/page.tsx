"use client";

import { useEffect, useState } from "react";
import { getDashboardOverview, getUsageData } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { formatCost, formatNumber } from "@/lib/utils";
import { Activity, DollarSign, Zap, Bot, TrendingDown, Database } from "lucide-react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";

interface Overview {
  total_requests_24h: number;
  total_cost_24h: number;
  cache_hit_rate: number;
  active_models: number;
  total_requests: number;
  total_cost: number;
  tokens_saved: number;
  cost_saved: number;
}

export default function DashboardPage() {
  const [overview, setOverview] = useState<Overview | null>(null);
  const [usage, setUsage] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([getDashboardOverview(), getUsageData("7d")])
      .then(([ov, us]) => { setOverview(ov); setUsage(us || []); })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Card key={i}>
              <CardHeader className="pb-2"><div className="h-4 w-24 bg-muted rounded animate-pulse" /></CardHeader>
              <CardContent><div className="h-7 w-16 bg-muted rounded animate-pulse" /></CardContent>
            </Card>
          ))}
        </div>
        <Card>
          <CardContent className="pt-6"><div className="h-[300px] bg-muted/50 rounded-lg animate-pulse" /></CardContent>
        </Card>
      </div>
    );
  }

  const stats = [
    { title: "Requests (24h)", value: formatNumber(overview?.total_requests_24h ?? 0), desc: `${formatNumber(overview?.total_requests ?? 0)} total`, icon: Activity },
    { title: "Cost (24h)", value: formatCost(overview?.total_cost_24h ?? 0), desc: `${formatCost(overview?.total_cost ?? 0)} total`, icon: DollarSign },
    { title: "Cache Hit Rate", value: `${(overview?.cache_hit_rate ?? 0).toFixed(1)}%`, desc: "Semantic similarity", icon: Zap },
    { title: "Active Models", value: overview?.active_models?.toString() ?? "0", desc: "Configured providers", icon: Bot },
  ];

  const savingsStats = [
    { title: "Tokens Saved", value: formatNumber(overview?.tokens_saved ?? 0), desc: "From cache hits", icon: TrendingDown },
    { title: "Cost Saved", value: formatCost(overview?.cost_saved ?? 0), desc: "Cache savings", icon: Database },
  ];

  return (
    <div className="space-y-6">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => (
          <Card key={stat.title}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">{stat.title}</CardTitle>
              <stat.icon className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stat.value}</div>
              <p className="text-xs text-muted-foreground">{stat.desc}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        {savingsStats.map((stat) => (
          <Card key={stat.title}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">{stat.title}</CardTitle>
              <stat.icon className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stat.value}</div>
              <p className="text-xs text-muted-foreground">{stat.desc}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Usage Overview</CardTitle>
          <CardDescription>Requests and cost over the last 7 days</CardDescription>
        </CardHeader>
        <CardContent>
          {usage.length > 0 ? (
            <ResponsiveContainer width="100%" height={320}>
              <AreaChart data={usage} margin={{ top: 8, right: 8, bottom: 0, left: -12 }}>
                <defs>
                  <linearGradient id="gradReq" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="hsl(197, 37%, 24%)" stopOpacity={0.2} />
                    <stop offset="100%" stopColor="hsl(197, 37%, 24%)" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="gradCost" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="hsl(173, 58%, 39%)" stopOpacity={0.2} />
                    <stop offset="100%" stopColor="hsl(173, 58%, 39%)" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="hsl(240 5.9% 90%)" vertical={false} />
                <XAxis
                  dataKey="date"
                  tick={{ fontSize: 12 }}
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={(v) => new Date(v).toLocaleDateString("en", { month: "short", day: "numeric" })}
                />
                <YAxis tick={{ fontSize: 12 }} tickLine={false} axisLine={false} />
                <Tooltip
                  contentStyle={{
                    backgroundColor: "hsl(0 0% 100%)",
                    borderColor: "hsl(240 5.9% 90%)",
                    borderRadius: "8px",
                    fontSize: "12px",
                  }}
                />
                <Area type="monotone" dataKey="requests" stroke="hsl(197, 37%, 24%)" fill="url(#gradReq)" strokeWidth={2} name="Requests" dot={false} />
                <Area type="monotone" dataKey="total_cost" stroke="hsl(173, 58%, 39%)" fill="url(#gradCost)" strokeWidth={2} name="Cost ($)" dot={false} />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex flex-col items-center justify-center h-[320px] text-muted-foreground">
              <Activity className="h-10 w-10 mb-3 opacity-20" />
              <p className="text-sm font-medium">No usage data yet</p>
              <p className="text-xs mt-1">Start sending requests to see trends</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
