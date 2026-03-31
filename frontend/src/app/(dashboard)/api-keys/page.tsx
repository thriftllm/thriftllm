"use client";

import { useEffect, useState, useCallback } from "react";
import { listAPIKeys, createAPIKey, deleteAPIKey, toggleAPIKey } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
  DialogFooter,
  DialogDescription,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Plus, Trash2, Copy, Check, Key, Shield } from "lucide-react";
import { timeAgo } from "@/lib/utils";

interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
  is_active: boolean;
  rate_limit_rpm: number;
  last_used_at: string | null;
  created_at: string;
}

export default function APIKeysPage() {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [name, setName] = useState("");
  const [rateLimit, setRateLimit] = useState(60);
  const [saving, setSaving] = useState(false);
  const [newKey, setNewKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const load = useCallback(async () => {
    try {
      const data = await listAPIKeys();
      setKeys(data || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleCreate = async () => {
    setSaving(true);
    try {
      const res = await createAPIKey({ name, rate_limit_rpm: rateLimit });
      setNewKey(res.key);
      load();
    } catch (err: any) {
      alert(err.message);
    } finally {
      setSaving(false);
    }
  };

  const handleCopy = () => {
    if (newKey) {
      navigator.clipboard.writeText(newKey);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handleCloseDialog = () => {
    setDialogOpen(false);
    setNewKey(null);
    setName("");
    setRateLimit(60);
    setCopied(false);
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Delete this API key? This cannot be undone.")) return;
    try {
      await deleteAPIKey(id);
      load();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleToggle = async (id: string, active: boolean) => {
    try {
      await toggleAPIKey(id, active);
      setKeys((prev) =>
        prev.map((k) => (k.id === id ? { ...k, is_active: active } : k))
      );
    } catch (err: any) {
      alert(err.message);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold tracking-tight">API Keys</h2>
          <p className="text-sm text-muted-foreground">Manage access tokens for the proxy endpoint.</p>
        </div>
        <Dialog open={dialogOpen} onOpenChange={(open) => open ? setDialogOpen(true) : handleCloseDialog()}>
          <DialogTrigger asChild>
            <Button size="sm" onClick={() => { setNewKey(null); setDialogOpen(true); }}>
              <Plus className="h-4 w-4 mr-1.5" /> Create Key
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>{newKey ? "Key Created" : "Create API Key"}</DialogTitle>
              {!newKey && <DialogDescription>Generate a new API key for your application.</DialogDescription>}
            </DialogHeader>
            {newKey ? (
              <div className="space-y-4 py-4">
                <div className="rounded-md border bg-muted/50 p-3 text-sm text-muted-foreground">
                  <Shield className="h-4 w-4 inline mr-1.5 -mt-0.5" />
                  Copy this key now. You won&apos;t be able to see it again.
                </div>
                <div className="flex items-center gap-2">
                  <Input value={newKey} readOnly className="font-mono text-xs" />
                  <Button size="icon" variant="outline" onClick={handleCopy}>
                    {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                  </Button>
                </div>
                <DialogFooter>
                  <Button onClick={handleCloseDialog}>Done</Button>
                </DialogFooter>
              </div>
            ) : (
              <div className="space-y-4 py-4">
                <div className="grid gap-2">
                  <Label>Name</Label>
                  <Input placeholder="My Application" value={name} onChange={(e) => setName(e.target.value)} />
                </div>
                <div className="grid gap-2">
                  <Label>Rate Limit (requests/min)</Label>
                  <Input type="number" value={rateLimit} onChange={(e) => setRateLimit(Number(e.target.value))} />
                </div>
                <DialogFooter>
                  <Button variant="outline" onClick={handleCloseDialog}>Cancel</Button>
                  <Button onClick={handleCreate} disabled={saving || !name}>
                    {saving ? "Creating..." : "Create"}
                  </Button>
                </DialogFooter>
              </div>
            )}
          </DialogContent>
        </Dialog>
      </div>

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-6 space-y-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="h-12 bg-muted/50 rounded animate-pulse" />
              ))}
            </div>
          ) : keys.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 px-6 text-center">
              <Key className="h-10 w-10 text-muted-foreground/30 mb-4" />
              <p className="font-medium mb-1">No API keys yet</p>
              <p className="text-sm text-muted-foreground mb-4">
                Create a key to start sending requests through the proxy.
              </p>
              <Button size="sm" onClick={() => { setNewKey(null); setDialogOpen(true); }}>
                <Plus className="h-4 w-4 mr-1.5" /> Create Key
              </Button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Key</TableHead>
                  <TableHead>Rate Limit</TableHead>
                  <TableHead>Last Used</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead>Active</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {keys.map((k) => (
                  <TableRow key={k.id} className="group">
                    <TableCell className="font-medium text-sm">{k.name}</TableCell>
                    <TableCell>
                      <code className="text-xs bg-muted px-2 py-0.5 rounded font-mono">{k.key_prefix}...</code>
                    </TableCell>
                    <TableCell>
                      <span className="text-sm tabular-nums">{k.rate_limit_rpm} <span className="text-muted-foreground text-xs">rpm</span></span>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {k.last_used_at ? timeAgo(k.last_used_at) : <span className="text-muted-foreground/50">Never</span>}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">{timeAgo(k.created_at)}</TableCell>
                    <TableCell>
                      <Switch checked={k.is_active} onCheckedChange={(v) => handleToggle(k.id, v)} />
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        size="icon"
                        variant="ghost"
                        className="h-8 w-8 text-destructive hover:text-destructive opacity-0 group-hover:opacity-100 transition-opacity"
                        onClick={() => handleDelete(k.id)}
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
