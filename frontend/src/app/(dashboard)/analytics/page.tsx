"use client";

import { useEffect, useState } from "react";
import { getUsageData, getModelBreakdown } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { formatCost, formatNumber } from "@/lib/utils";
import { DollarSign, Activity, Layers } from "lucide-react";
import {
  BarChart,
  Bar,
  AreaChart,
  Area,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";

const CHART_COLORS = [
  "hsl(12, 76%, 61%)",
  "hsl(173, 58%, 39%)",
  "hsl(197, 37%, 24%)",
  "hsl(43, 74%, 66%)",
  "hsl(27, 87%, 67%)",
  "hsl(240, 5%, 64%)",
  "hsl(322, 40%, 50%)",
  "hsl(60, 40%, 50%)",
];

const tooltipStyle = {
  backgroundColor: "hsl(0 0% 100%)",
  borderColor: "hsl(240 5.9% 90%)",
  borderRadius: "8px",
  fontSize: "12px",
};

export default function AnalyticsPage() {
  const [range, setRange] = useState("7d");
  const [usage, setUsage] = useState<any[]>([]);
  const [models, setModels] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    Promise.all([getUsageData(range), getModelBreakdown(range)])
      .then(([u, m]) => { setUsage(u || []); setModels(m || []); })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, [range]);

  const totalCost = models.reduce((acc: number, m: any) => acc + (m.cost || 0), 0);
  const totalRequests = models.reduce((acc: number, m: any) => acc + (m.requests || 0), 0);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold tracking-tight">Analytics</h2>
          <p className="text-sm text-muted-foreground">Usage trends and cost analysis.</p>
        </div>
        <Tabs value={range} onValueChange={setRange}>
          <TabsList className="h-8">
            <TabsTrigger value="7d" className="text-xs px-3 h-7">7D</TabsTrigger>
            <TabsTrigger value="30d" className="text-xs px-3 h-7">30D</TabsTrigger>
            <TabsTrigger value="90d" className="text-xs px-3 h-7">90D</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      {loading ? (
        <div className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Card key={i}><CardHeader className="pb-2"><div className="h-4 w-24 bg-muted rounded animate-pulse" /></CardHeader><CardContent><div className="h-7 w-16 bg-muted rounded animate-pulse" /></CardContent></Card>
            ))}
          </div>
          <Card><CardContent className="pt-6"><div className="h-[300px] bg-muted/50 rounded-lg animate-pulse" /></CardContent></Card>
        </div>
      ) : (
        <>
          <div className="grid gap-4 sm:grid-cols-3">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Total Cost</CardTitle>
                <DollarSign className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{formatCost(totalCost)}</div>
                <p className="text-xs text-muted-foreground">For selected period</p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Total Requests</CardTitle>
                <Activity className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{formatNumber(totalRequests)}</div>
                <p className="text-xs text-muted-foreground">For selected period</p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Models Used</CardTitle>
                <Layers className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{models.length}</div>
                <p className="text-xs text-muted-foreground">Unique providers</p>
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Cost Over Time</CardTitle>
              <CardDescription>Daily cost breakdown</CardDescription>
            </CardHeader>
            <CardContent>
              {usage.length > 0 ? (
                <ResponsiveContainer width="100%" height={300}>
                  <AreaChart data={usage} margin={{ top: 8, right: 8, bottom: 0, left: -12 }}>
                    <defs>
                      <linearGradient id="gradCostAnalytics" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor="hsl(173, 58%, 39%)" stopOpacity={0.2} />
                        <stop offset="100%" stopColor="hsl(173, 58%, 39%)" stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke="hsl(240 5.9% 90%)" vertical={false} />
                    <XAxis dataKey="date" tick={{ fontSize: 12 }} tickLine={false} axisLine={false} tickFormatter={(v) => new Date(v).toLocaleDateString("en", { month: "short", day: "numeric" })} />
                    <YAxis tick={{ fontSize: 12 }} tickLine={false} axisLine={false} tickFormatter={(v) => `$${v}`} />
                    <Tooltip formatter={(value: number) => formatCost(value)} contentStyle={tooltipStyle} />
                    <Area type="monotone" dataKey="total_cost" stroke="hsl(173, 58%, 39%)" fill="url(#gradCostAnalytics)" strokeWidth={2} name="Cost" dot={false} />
                  </AreaChart>
                </ResponsiveContainer>
              ) : (
                <div className="flex flex-col items-center justify-center h-[300px] text-muted-foreground">
                  <DollarSign className="h-10 w-10 mb-3 opacity-20" />
                  <p className="text-sm">No cost data available</p>
                </div>
              )}
            </CardContent>
          </Card>

          <div className="grid gap-4 md:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>Requests Over Time</CardTitle>
                <CardDescription>Daily request volume</CardDescription>
              </CardHeader>
              <CardContent>
                {usage.length > 0 ? (
                  <ResponsiveContainer width="100%" height={280}>
                    <BarChart data={usage} margin={{ top: 8, right: 8, bottom: 0, left: -12 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke="hsl(240 5.9% 90%)" vertical={false} />
                      <XAxis dataKey="date" tick={{ fontSize: 11 }} tickLine={false} axisLine={false} tickFormatter={(v) => new Date(v).toLocaleDateString("en", { month: "short", day: "numeric" })} />
                      <YAxis tick={{ fontSize: 11 }} tickLine={false} axisLine={false} />
                      <Tooltip contentStyle={tooltipStyle} />
                      <Bar dataKey="requests" fill="hsl(197, 37%, 24%)" radius={[4, 4, 0, 0]} name="Requests" maxBarSize={40} />
                    </BarChart>
                  </ResponsiveContainer>
                ) : (
                  <div className="flex items-center justify-center h-[280px] text-muted-foreground text-sm">No data available</div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Model Breakdown</CardTitle>
                <CardDescription>Request distribution by model</CardDescription>
              </CardHeader>
              <CardContent>
                {models.length > 0 ? (
                  <div className="space-y-4">
                    <ResponsiveContainer width="100%" height={180}>
                      <PieChart>
                        <Pie
                          data={models}
                          dataKey="requests"
                          nameKey="model"
                          cx="50%"
                          cy="50%"
                          innerRadius={45}
                          outerRadius={75}
                          paddingAngle={2}
                        >
                          {models.map((_, i) => (
                            <Cell key={i} fill={CHART_COLORS[i % CHART_COLORS.length]} />
                          ))}
                        </Pie>
                        <Tooltip contentStyle={tooltipStyle} />
                      </PieChart>
                    </ResponsiveContainer>
                    <div className="space-y-2 px-2">
                      {models.map((m: any, i: number) => (
                        <div key={m.model} className="flex items-center justify-between text-sm">
                          <div className="flex items-center gap-2 min-w-0">
                            <div className="h-2.5 w-2.5 rounded-full shrink-0" style={{ backgroundColor: CHART_COLORS[i % CHART_COLORS.length] }} />
                            <span className="truncate text-xs font-medium">{m.model}</span>
                          </div>
                          <div className="flex gap-4 text-muted-foreground text-xs tabular-nums shrink-0">
                            <span>{formatNumber(m.requests)} req</span>
                            <span className="w-16 text-right">{formatCost(m.cost)}</span>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : (
                  <div className="flex items-center justify-center h-[280px] text-muted-foreground text-sm">No data available</div>
                )}
              </CardContent>
            </Card>
          </div>
        </>
      )}
    </div>
  );
}
