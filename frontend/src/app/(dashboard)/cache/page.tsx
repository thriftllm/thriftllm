"use client";

import { useEffect, useState } from "react";
import { getCacheStats, flushCache } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { formatNumber } from "@/lib/utils";
import { Database, Zap, Trash2, TrendingUp, Activity, BarChart3 } from "lucide-react";
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from "recharts";

interface CacheStatsResponse {
  entry_count: number;
  overview: {
    total_requests: number;
    cache_hits: number;
    cache_misses: number;
    hit_rate: number;
    tokens_saved: number;
    cost_saved: number;
  };
}

const tooltipStyle = {
  backgroundColor: "hsl(0 0% 100%)",
  borderColor: "hsl(240 5.9% 90%)",
  borderRadius: "8px",
  fontSize: "12px",
};

export default function CachePage() {
  const [stats, setStats] = useState<CacheStatsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [flushing, setFlushing] = useState(false);

  const load = async () => {
    try {
      const data = await getCacheStats();
      setStats(data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const handleFlush = async () => {
    if (!confirm("Flush all cached responses? This cannot be undone.")) return;
    setFlushing(true);
    try {
      await flushCache();
      load();
    } catch (err: any) {
      alert(err.message);
    } finally {
      setFlushing(false);
    }
  };

  const hitRate = stats?.overview?.hit_rate ?? 0;
  const pieData = [
    { name: "Hits", value: stats?.overview?.cache_hits ?? 0 },
    { name: "Misses", value: stats?.overview?.cache_misses ?? 0 },
  ];
  const hasCacheActivity = (pieData[0].value + pieData[1].value) > 0;

  const statItems = [
    { title: "Hit Rate", value: `${hitRate.toFixed(1)}%`, icon: Zap },
    { title: "Total Requests", value: formatNumber(stats?.overview?.total_requests ?? 0), icon: Activity },
    { title: "Tokens Saved", value: formatNumber(stats?.overview?.tokens_saved ?? 0), icon: BarChart3 },
    { title: "Cost Saved", value: `$${(stats?.overview?.cost_saved ?? 0).toFixed(4)}`, icon: TrendingUp },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold tracking-tight">Cache</h2>
          <p className="text-sm text-muted-foreground">Semantic cache performance and management.</p>
        </div>
        <Button variant="outline" size="sm" onClick={handleFlush} disabled={flushing} className="text-destructive hover:text-destructive">
          <Trash2 className="h-3.5 w-3.5 mr-1.5" />
          {flushing ? "Flushing..." : "Flush Cache"}
        </Button>
      </div>

      {loading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Card key={i}><CardHeader className="pb-2"><div className="h-4 w-24 bg-muted rounded animate-pulse" /></CardHeader><CardContent><div className="h-7 w-16 bg-muted rounded animate-pulse" /></CardContent></Card>
          ))}
        </div>
      ) : (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {statItems.map((stat) => (
              <Card key={stat.title}>
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">{stat.title}</CardTitle>
                  <stat.icon className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{stat.value}</div>
                </CardContent>
              </Card>
            ))}
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle>Cache Distribution</CardTitle>
                    <CardDescription>Hits vs misses breakdown</CardDescription>
                  </div>
                  {stats && (
                    <span className="text-xs text-muted-foreground tabular-nums">
                      {stats.entry_count} entries
                    </span>
                  )}
                </div>
              </CardHeader>
              <CardContent>
                {hasCacheActivity ? (
                  <div className="flex flex-col items-center">
                    <ResponsiveContainer width="100%" height={220}>
                      <PieChart>
                        <Pie
                          data={pieData}
                          cx="50%"
                          cy="50%"
                          innerRadius={60}
                          outerRadius={85}
                          paddingAngle={3}
                          dataKey="value"
                        >
                          <Cell fill="hsl(173, 58%, 39%)" />
                          <Cell fill="hsl(240, 3.8%, 46.1%)" />
                        </Pie>
                        <Tooltip contentStyle={tooltipStyle} />
                      </PieChart>
                    </ResponsiveContainer>
                    <div className="flex gap-6 text-sm">
                      <div className="flex items-center gap-2">
                        <div className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: "hsl(173, 58%, 39%)" }} />
                        <span className="text-muted-foreground">Hits</span>
                        <span className="font-medium tabular-nums">{stats?.overview?.cache_hits ?? 0}</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <div className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: "hsl(240, 3.8%, 46.1%)" }} />
                        <span className="text-muted-foreground">Misses</span>
                        <span className="font-medium tabular-nums">{stats?.overview?.cache_misses ?? 0}</span>
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className="flex flex-col items-center justify-center h-[250px] text-muted-foreground">
                    <Database className="h-10 w-10 mb-3 opacity-20" />
                    <p className="text-sm font-medium">No cache activity yet</p>
                    <p className="text-xs mt-1">Cache hits appear as requests come in</p>
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>How Semantic Caching Works</CardTitle>
                <CardDescription>Vector similarity search for response reuse</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-3">
                  {[
                    { step: "1", title: "Embed", desc: "Request is converted to a 64-dimensional vector embedding" },
                    { step: "2", title: "Search", desc: "Redis RediSearch performs KNN vector similarity search" },
                    { step: "3", title: "Match", desc: "If cosine similarity >= 95%, cached response is returned" },
                    { step: "4", title: "Expire", desc: "Cache entries automatically expire after 24 hours" },
                  ].map((item) => (
                    <div key={item.step} className="flex gap-3">
                      <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-md border text-[11px] font-medium">
                        {item.step}
                      </div>
                      <div>
                        <p className="text-sm font-medium">{item.title}</p>
                        <p className="text-xs text-muted-foreground">{item.desc}</p>
                      </div>
                    </div>
                  ))}
                </div>
                <div className="rounded-md border p-3 text-xs text-muted-foreground">
                  <strong className="text-foreground">Note:</strong> Streaming requests and requests with temperature &gt; 0.5 bypass the cache.
                </div>
              </CardContent>
            </Card>
          </div>
        </>
      )}
    </div>
  );
}
