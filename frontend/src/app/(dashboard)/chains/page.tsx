"use client";

import { useEffect, useState } from "react";
import {
  listFallbackChains,
  createFallbackChain,
  updateFallbackChain,
  deleteFallbackChain,
  listModels,
} from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import { Plus, Trash2, Pencil, GitBranch, ArrowDown, GripVertical } from "lucide-react";

interface FallbackChain {
  id: string;
  name: string;
  model_config_ids: string[];
  tag_selector?: string | null;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

interface ModelConfig {
  id: string;
  provider: string;
  provider_model: string;
  display_name: string;
  is_active: boolean;
  priority: number;
  tags: string[];
}

export default function ChainsPage() {
  const [chains, setChains] = useState<FallbackChain[]>([]);
  const [models, setModels] = useState<ModelConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingChain, setEditingChain] = useState<FallbackChain | null>(null);

  const [name, setName] = useState("");
  const [tagSelector, setTagSelector] = useState("");
  const [isDefault, setIsDefault] = useState(false);
  const [selectedModelIds, setSelectedModelIds] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);

  const load = async () => {
    try {
      const [chainsData, modelsData] = await Promise.all([
        listFallbackChains(),
        listModels(),
      ]);
      setChains(chainsData);
      setModels(modelsData);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const activeModels = models.filter((m) => m.is_active);

  const resetForm = () => {
    setName("");
    setTagSelector("");
    setIsDefault(false);
    setSelectedModelIds([]);
    setEditingChain(null);
  };

  const openCreate = () => { resetForm(); setDialogOpen(true); };

  const openEdit = (chain: FallbackChain) => {
    setEditingChain(chain);
    setName(chain.name);
    setTagSelector(chain.tag_selector || "");
    setIsDefault(chain.is_default);
    setSelectedModelIds(chain.model_config_ids || []);
    setDialogOpen(true);
  };

  const handleSave = async () => {
    if (!name || selectedModelIds.length === 0) return;
    setSaving(true);
    try {
      const payload = {
        name,
        model_config_ids: selectedModelIds,
        tag_selector: tagSelector || null,
        is_default: isDefault,
      };
      if (editingChain) {
        await updateFallbackChain(editingChain.id, payload);
      } else {
        await createFallbackChain(payload);
      }
      setDialogOpen(false);
      resetForm();
      load();
    } catch (err: any) {
      alert(err.message);
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Delete this fallback chain?")) return;
    try {
      await deleteFallbackChain(id);
      load();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const toggleModel = (modelId: string) => {
    setSelectedModelIds((prev) =>
      prev.includes(modelId)
        ? prev.filter((id) => id !== modelId)
        : [...prev, modelId]
    );
  };

  const moveModel = (index: number, direction: "up" | "down") => {
    setSelectedModelIds((prev) => {
      const next = [...prev];
      const target = direction === "up" ? index - 1 : index + 1;
      if (target < 0 || target >= next.length) return prev;
      [next[index], next[target]] = [next[target], next[index]];
      return next;
    });
  };

  const getModelById = (id: string) => models.find((m) => m.id === id);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold tracking-tight">Fallback Chains</h2>
          <p className="text-sm text-muted-foreground">Configure model fallback priority ordering.</p>
        </div>
        <Dialog open={dialogOpen} onOpenChange={(open) => { if (!open) { setDialogOpen(false); resetForm(); } }}>
          <DialogTrigger asChild>
            <Button size="sm" onClick={openCreate}>
              <Plus className="h-4 w-4 mr-1.5" /> Create Chain
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-lg max-h-[85vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>
                {editingChain ? "Edit Fallback Chain" : "Create Fallback Chain"}
              </DialogTitle>
              <DialogDescription>
                Define the order in which models are tried when a request fails.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="grid gap-2">
                <Label>Chain Name</Label>
                <Input
                  placeholder="e.g. Production Chain"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </div>
              <div className="grid gap-2">
                <Label>Tag Selector</Label>
                <Input
                  placeholder="e.g. production (optional)"
                  value={tagSelector}
                  onChange={(e) => setTagSelector(e.target.value)}
                />
                <p className="text-[11px] text-muted-foreground">
                  Requests with this tag in X-Thrift-Tags header will use this chain.
                </p>
              </div>
              <div className="flex items-center justify-between rounded-md border p-3">
                <div>
                  <Label>Default Chain</Label>
                  <p className="text-[11px] text-muted-foreground">Used when no tag matches</p>
                </div>
                <Switch checked={isDefault} onCheckedChange={setIsDefault} />
              </div>

              <div className="space-y-2">
                <Label>
                  Models ({selectedModelIds.length} selected)
                </Label>
                <p className="text-[11px] text-muted-foreground">Order determines fallback priority.</p>

                {selectedModelIds.length > 0 && (
                  <div className="space-y-1 rounded-md border p-2">
                    {selectedModelIds.map((id, i) => {
                      const m = getModelById(id);
                      if (!m) return null;
                      return (
                        <div key={id} className="flex items-center gap-2 rounded-md bg-muted/50 px-2 py-1.5 text-sm">
                          <GripVertical className="h-3.5 w-3.5 text-muted-foreground/40" />
                          <span className="text-xs font-medium text-muted-foreground w-5 tabular-nums">{i + 1}.</span>
                          <Badge variant="outline" className="text-[10px] font-normal">{m.provider}</Badge>
                          <span className="text-xs flex-1 truncate">{m.display_name}</span>
                          <div className="flex items-center gap-0.5">
                            <Button variant="ghost" size="sm" className="h-6 w-6 p-0" disabled={i === 0} onClick={() => moveModel(i, "up")}>
                              <ArrowDown className="h-3 w-3 rotate-180" />
                            </Button>
                            <Button variant="ghost" size="sm" className="h-6 w-6 p-0" disabled={i === selectedModelIds.length - 1} onClick={() => moveModel(i, "down")}>
                              <ArrowDown className="h-3 w-3" />
                            </Button>
                            <Button variant="ghost" size="sm" className="h-6 w-6 p-0 text-destructive hover:text-destructive" onClick={() => toggleModel(id)}>
                              <Trash2 className="h-3 w-3" />
                            </Button>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}

                <div className="space-y-1 max-h-48 overflow-y-auto">
                  {activeModels
                    .filter((m) => !selectedModelIds.includes(m.id))
                    .map((m) => (
                      <button
                        key={m.id}
                        className="w-full flex items-center gap-2 rounded-md border border-dashed px-2 py-1.5 text-sm hover:bg-muted/50 transition-colors text-left cursor-pointer"
                        onClick={() => toggleModel(m.id)}
                      >
                        <Plus className="h-3 w-3 text-muted-foreground" />
                        <Badge variant="outline" className="text-[10px] font-normal">{m.provider}</Badge>
                        <span className="text-xs flex-1 truncate">{m.display_name}</span>
                        <span className="text-[10px] text-muted-foreground">{m.provider_model}</span>
                      </button>
                    ))}
                  {activeModels.filter((m) => !selectedModelIds.includes(m.id)).length === 0 && (
                    <p className="text-xs text-muted-foreground text-center py-2">All active models selected</p>
                  )}
                </div>
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => { setDialogOpen(false); resetForm(); }}>Cancel</Button>
              <Button onClick={handleSave} disabled={saving || !name || selectedModelIds.length === 0}>
                {saving ? "Saving..." : editingChain ? "Update" : "Create"}
              </Button>
            </DialogFooter>
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
          ) : chains.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 text-center">
              <GitBranch className="h-10 w-10 text-muted-foreground/30 mb-4" />
              <p className="font-medium mb-1">No fallback chains</p>
              <p className="text-sm text-muted-foreground mb-4">
                Create a chain to define model fallback order.
              </p>
              <Button size="sm" onClick={openCreate}>
                <Plus className="h-4 w-4 mr-1.5" /> Create Chain
              </Button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Models</TableHead>
                  <TableHead>Tag</TableHead>
                  <TableHead>Default</TableHead>
                  <TableHead className="w-[80px]" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {chains.map((chain) => (
                  <TableRow key={chain.id} className="group">
                    <TableCell>
                      <span className="font-medium text-sm">{chain.name}</span>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1 flex-wrap">
                        {(chain.model_config_ids || []).map((id, i) => {
                          const m = getModelById(id);
                          return (
                            <span key={id} className="flex items-center gap-1">
                              {i > 0 && <span className="text-[10px] text-muted-foreground">&rarr;</span>}
                              <Badge variant="outline" className="text-[10px] font-normal">
                                {m ? m.display_name : "deleted"}
                              </Badge>
                            </span>
                          );
                        })}
                      </div>
                    </TableCell>
                    <TableCell>
                      {chain.tag_selector ? (
                        <Badge variant="secondary" className="text-[10px]">{chain.tag_selector}</Badge>
                      ) : (
                        <span className="text-xs text-muted-foreground">&mdash;</span>
                      )}
                    </TableCell>
                    <TableCell>
                      {chain.is_default ? (
                        <Badge variant="secondary" className="text-[10px]">Default</Badge>
                      ) : (
                        <span className="text-xs text-muted-foreground">&mdash;</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                        <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={() => openEdit(chain)}>
                          <Pencil className="h-3.5 w-3.5" />
                        </Button>
                        <Button variant="ghost" size="sm" className="h-7 w-7 p-0 text-destructive hover:text-destructive" onClick={() => handleDelete(chain.id)}>
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium">How Fallback Chains Work</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3 text-sm">
            {[
              { step: "1", title: "Request arrives", desc: "ThriftLLM receives a /v1/chat/completions request." },
              { step: "2", title: "Chain resolution", desc: "Matching tag selector chain is used, or the default chain." },
              { step: "3", title: "Ordered fallback", desc: "Models are tried in chain order until one succeeds." },
              { step: "4", title: "Transparent to caller", desc: "Response headers indicate which model served the request." },
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
        </CardContent>
      </Card>
    </div>
  );
}
