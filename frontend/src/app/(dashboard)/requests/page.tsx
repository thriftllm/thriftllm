"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import { listRequests } from "@/lib/api";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  Search,
  FileText,
  Zap,
} from "lucide-react";
import { formatCost, formatLatency, timeAgo } from "@/lib/utils";

interface RequestLog {
  id: string;
  requested_model: string;
  actual_model: string;
  actual_provider: string;
  status_code: number;
  latency_ms: number;
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
  total_cost_cents: number;
  cache_hit: boolean;
  cache_similarity?: number;
  fallback_depth: number;
  is_streaming: boolean;
  created_at: string;
}

const PAGE_SIZES = [10, 25, 50, 100];

export default function RequestsPage() {
  const [logs, setLogs] = useState<RequestLog[]>([]);
  const [total, setTotal] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [page, setPage] = useState(1);
  const [limit, setLimit] = useState(25);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [providerFilter, setProviderFilter] = useState("all");
  const debounceTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Debounce search input
  useEffect(() => {
    if (debounceTimer.current) clearTimeout(debounceTimer.current);
    debounceTimer.current = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 400);
    return () => {
      if (debounceTimer.current) clearTimeout(debounceTimer.current);
    };
  }, [search]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = {
        page: page.toString(),
        limit: limit.toString(),
      };
      if (debouncedSearch) params.search = debouncedSearch;
      if (statusFilter === "success") params.status = "200";
      if (statusFilter === "error") params.status = "500";
      if (statusFilter === "cached") params.cache = "true";
      if (providerFilter !== "all") params.provider = providerFilter;

      const data = await listRequests(params);
      setLogs(data.logs || []);
      setTotal(data.total);
      setTotalPages(data.total_pages || Math.ceil(data.total / limit));
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, [page, limit, debouncedSearch, statusFilter, providerFilter]);

  useEffect(() => {
    load();
  }, [load]);

  const getStatusBadge = (code: number, cacheHit: boolean) => {
    if (cacheHit)
      return (
        <Badge variant="success" className="gap-1">
          <Zap className="h-3 w-3" />
          Cached
        </Badge>
      );
    if (code >= 200 && code < 300)
      return <Badge variant="success">{code}</Badge>;
    if (code >= 400) return <Badge variant="destructive">{code}</Badge>;
    return <Badge variant="secondary">{code}</Badge>;
  };

  // Generate page numbers to display
  const getPageNumbers = (): (number | "ellipsis")[] => {
    if (totalPages <= 7) {
      return Array.from({ length: totalPages }, (_, i) => i + 1);
    }
    const pages: (number | "ellipsis")[] = [1];
    if (page > 3) pages.push("ellipsis");
    const start = Math.max(2, page - 1);
    const end = Math.min(totalPages - 1, page + 1);
    for (let i = start; i <= end; i++) pages.push(i);
    if (page < totalPages - 2) pages.push("ellipsis");
    if (totalPages > 1) pages.push(totalPages);
    return pages;
  };

  const handlePageSizeChange = (newSize: string) => {
    setLimit(Number(newSize));
    setPage(1);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold tracking-tight">Requests</h2>
          <p className="text-sm text-muted-foreground">
            View all proxied LLM request logs.
          </p>
        </div>
        {total > 0 && (
          <span className="text-xs text-muted-foreground tabular-nums">
            {total.toLocaleString()} total request{total !== 1 ? "s" : ""}
          </span>
        )}
      </div>

      {/* Filters row */}
      <div className="flex flex-wrap gap-3 items-center">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search model, provider..."
            className="pl-9 h-9"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <Select
          value={statusFilter}
          onValueChange={(v) => {
            setStatusFilter(v);
            setPage(1);
          }}
        >
          <SelectTrigger className="w-36 h-9">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Status</SelectItem>
            <SelectItem value="success">Success</SelectItem>
            <SelectItem value="error">Error</SelectItem>
            <SelectItem value="cached">Cached</SelectItem>
          </SelectContent>
        </Select>
        <Select
          value={providerFilter}
          onValueChange={(v) => {
            setProviderFilter(v);
            setPage(1);
          }}
        >
          <SelectTrigger className="w-36 h-9">
            <SelectValue placeholder="Provider" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Providers</SelectItem>
            <SelectItem value="openai">OpenAI</SelectItem>
            <SelectItem value="anthropic">Anthropic</SelectItem>
            <SelectItem value="google">Google</SelectItem>
            <SelectItem value="custom_openai">Custom</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Table */}
      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-6 space-y-3">
              {Array.from({ length: 5 }).map((_, i) => (
                <div
                  key={i}
                  className="h-10 bg-muted/50 rounded animate-pulse"
                />
              ))}
            </div>
          ) : logs.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 px-6 text-center">
              <FileText className="h-10 w-10 text-muted-foreground/30 mb-4" />
              <p className="font-medium mb-1">No requests found</p>
              <p className="text-sm text-muted-foreground">
                {search || statusFilter !== "all" || providerFilter !== "all"
                  ? "Try adjusting your filters."
                  : "Requests will appear here as they come in."}
              </p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Time</TableHead>
                  <TableHead>Requested</TableHead>
                  <TableHead>Used</TableHead>
                  <TableHead>Provider</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Latency</TableHead>
                  <TableHead>Tokens</TableHead>
                  <TableHead>Cost</TableHead>
                  <TableHead>Fallback</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {logs.map((log) => (
                  <TableRow key={log.id}>
                    <TableCell className="text-xs text-muted-foreground whitespace-nowrap">
                      {timeAgo(log.created_at)}
                    </TableCell>
                    <TableCell>
                      <code className="text-xs bg-muted px-1.5 py-0.5 rounded font-mono">
                        {log.requested_model}
                      </code>
                    </TableCell>
                    <TableCell>
                      <span className="font-mono text-xs">
                        {log.actual_model}
                      </span>
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant="outline"
                        className="text-[10px] font-normal"
                      >
                        {log.actual_provider}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {getStatusBadge(log.status_code, log.cache_hit)}
                    </TableCell>
                    <TableCell className="text-sm tabular-nums">
                      {formatLatency(log.latency_ms)}
                    </TableCell>
                    <TableCell className="text-xs tabular-nums text-muted-foreground">
                      {(log.input_tokens + log.output_tokens).toLocaleString()}
                    </TableCell>
                    <TableCell className="text-sm tabular-nums">
                      {formatCost(log.total_cost_cents)}
                    </TableCell>
                    <TableCell>
                      {log.fallback_depth > 0 ? (
                        <Badge variant="warning" className="text-[10px]">
                          +{log.fallback_depth}
                        </Badge>
                      ) : (
                        <span className="text-muted-foreground/40">
                          &mdash;
                        </span>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Pagination controls */}
      {total > 0 && (
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <p className="text-xs text-muted-foreground tabular-nums">
              {((page - 1) * limit + 1).toLocaleString()}&ndash;
              {Math.min(page * limit, total).toLocaleString()} of{" "}
              {total.toLocaleString()}
            </p>
            <div className="flex items-center gap-1.5">
              <span className="text-xs text-muted-foreground">Rows</span>
              <Select
                value={limit.toString()}
                onValueChange={handlePageSizeChange}
              >
                <SelectTrigger className="w-[70px] h-8 text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {PAGE_SIZES.map((size) => (
                    <SelectItem key={size} value={size.toString()}>
                      {size}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="flex items-center gap-1">
            <Button
              variant="outline"
              size="sm"
              className="h-8 w-8 p-0"
              onClick={() => setPage(1)}
              disabled={page <= 1}
            >
              <ChevronsLeft className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="h-8 w-8 p-0"
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>

            {totalPages > 0 && (
              <div className="flex items-center gap-1 mx-1">
                {getPageNumbers().map((p, i) =>
                  p === "ellipsis" ? (
                    <span
                      key={`ellipsis-${i}`}
                      className="px-1 text-xs text-muted-foreground"
                    >
                      &hellip;
                    </span>
                  ) : (
                    <Button
                      key={p}
                      variant={page === p ? "default" : "outline"}
                      size="sm"
                      className="h-8 w-8 p-0 text-xs"
                      onClick={() => setPage(p)}
                    >
                      {p}
                    </Button>
                  )
                )}
              </div>
            )}

            <Button
              variant="outline"
              size="sm"
              className="h-8 w-8 p-0"
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages}
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="h-8 w-8 p-0"
              onClick={() => setPage(totalPages)}
              disabled={page >= totalPages}
            >
              <ChevronsRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
