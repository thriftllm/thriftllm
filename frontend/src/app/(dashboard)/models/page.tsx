"use client";

import { useEffect, useState, useCallback } from "react";
import {
  listModels,
  createModel,
  updateModel,
  deleteModel,
  toggleModel,
} from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
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
import { Plus, Trash2, Pencil, Bot } from "lucide-react";
import { formatCost } from "@/lib/utils";

const PROVIDERS = [
  "openai",
  "anthropic",
  "gemini",
  "groq",
  "together",
  "openrouter",
  "custom_openai",
];

interface ModelConfig {
  id: string;
  provider: string;
  provider_model: string;
  display_name: string;
  api_key_env_name: string;
  api_base_url: string | null;
  is_active: boolean;
  priority: number;
  input_cost_per_1k: number;
  output_cost_per_1k: number;
  tags: string[];
  created_at: string;
}

const emptyForm = {
  provider: "openai",
  provider_model: "",
  display_name: "",
  api_key_env_name: "",
  api_base_url: "",
  priority: 0,
  input_cost_per_1k: 0,
  output_cost_per_1k: 0,
  tags: "",
};

export default function ModelsPage() {
  const [models, setModels] = useState<ModelConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<ModelConfig | null>(null);
  const [form, setForm] = useState(emptyForm);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    try {
      const data = await listModels();
      setModels(data || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const openCreate = () => {
    setEditing(null);
    setForm(emptyForm);
    setDialogOpen(true);
  };

  const openEdit = (m: ModelConfig) => {
    setEditing(m);
    setForm({
      provider: m.provider,
      provider_model: m.provider_model,
      display_name: m.display_name,
      api_key_env_name: m.api_key_env_name || "",
      api_base_url: m.api_base_url || "",
      priority: m.priority,
      input_cost_per_1k: m.input_cost_per_1k,
      output_cost_per_1k: m.output_cost_per_1k,
      tags: (m.tags || []).join(", "),
    });
    setDialogOpen(true);
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      const payload = {
        ...form,
        api_base_url: form.api_base_url || null,
        priority: Number(form.priority),
        input_cost_per_1k: Number(form.input_cost_per_1k),
        output_cost_per_1k: Number(form.output_cost_per_1k),
        tags: form.tags
          .split(",")
          .map((t) => t.trim())
          .filter(Boolean),
      };
      if (editing) {
        await updateModel(editing.id, payload);
      } else {
        await createModel(payload);
      }
      setDialogOpen(false);
      load();
    } catch (err: any) {
      alert(err.message);
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Delete this model?")) return;
    try {
      await deleteModel(id);
      load();
    } catch (err: any) {
      alert(err.message);
    }
  };

  const handleToggle = async (id: string, active: boolean) => {
    try {
      await toggleModel(id, active);
      setModels((prev) =>
        prev.map((m) => (m.id === id ? { ...m, is_active: active } : m))
      );
    } catch (err: any) {
      alert(err.message);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold tracking-tight">Models</h2>
          <p className="text-sm text-muted-foreground">Manage your LLM provider configurations.</p>
        </div>
        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogTrigger asChild>
            <Button onClick={openCreate} size="sm">
              <Plus className="h-4 w-4 mr-1.5" /> Add Model
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-lg">
            <DialogHeader>
              <DialogTitle>{editing ? "Edit Model" : "Add Model"}</DialogTitle>
              <DialogDescription>
                {editing ? "Update the model configuration." : "Configure a new LLM provider model."}
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label>Provider</Label>
                <Select
                  value={form.provider}
                  onValueChange={(v) => setForm({ ...form, provider: v })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {PROVIDERS.map((p) => (
                      <SelectItem key={p} value={p}>
                        {p === "custom_openai" ? "Custom OpenAI" : p.charAt(0).toUpperCase() + p.slice(1)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label>Model Name</Label>
                <Input
                  placeholder="gpt-4o"
                  value={form.provider_model}
                  onChange={(e) => setForm({ ...form, provider_model: e.target.value })}
                />
              </div>
              <div className="grid gap-2">
                <Label>Display Name</Label>
                <Input
                  placeholder="GPT-4o (optional)"
                  value={form.display_name}
                  onChange={(e) => setForm({ ...form, display_name: e.target.value })}
                />
              </div>
              <div className="grid gap-2">
                <Label>API Key Env Variable</Label>
                <Input
                  placeholder="OPENAI_API_KEY"
                  value={form.api_key_env_name}
                  onChange={(e) => setForm({ ...form, api_key_env_name: e.target.value })}
                />
              </div>
              {form.provider === "custom_openai" && (
                <div className="grid gap-2">
                  <Label>Base URL</Label>
                  <Input
                    placeholder="https://api.example.com/v1"
                    value={form.api_base_url}
                    onChange={(e) => setForm({ ...form, api_base_url: e.target.value })}
                  />
                </div>
              )}
              <div className="grid grid-cols-3 gap-4">
                <div className="grid gap-2">
                  <Label>Priority</Label>
                  <Input
                    type="number"
                    value={form.priority}
                    onChange={(e) => setForm({ ...form, priority: Number(e.target.value) })}
                  />
                </div>
                <div className="grid gap-2">
                  <Label>Input $/1K</Label>
                  <Input
                    type="number"
                    step="0.0001"
                    value={form.input_cost_per_1k}
                    onChange={(e) => setForm({ ...form, input_cost_per_1k: Number(e.target.value) })}
                  />
                </div>
                <div className="grid gap-2">
                  <Label>Output $/1K</Label>
                  <Input
                    type="number"
                    step="0.0001"
                    value={form.output_cost_per_1k}
                    onChange={(e) => setForm({ ...form, output_cost_per_1k: Number(e.target.value) })}
                  />
                </div>
              </div>
              <div className="grid gap-2">
                <Label>Tags (comma-separated)</Label>
                <Input
                  placeholder="fast, cheap, coding"
                  value={form.tags}
                  onChange={(e) => setForm({ ...form, tags: e.target.value })}
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setDialogOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleSave} disabled={saving || !form.provider_model}>
                {saving ? "Saving..." : editing ? "Update" : "Create"}
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
          ) : models.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 px-6 text-center">
              <Bot className="h-10 w-10 text-muted-foreground/30 mb-4" />
              <p className="font-medium mb-1">No models configured</p>
              <p className="text-sm text-muted-foreground mb-4">
                Add your first LLM provider to start proxying requests.
              </p>
              <Button onClick={openCreate} size="sm">
                <Plus className="h-4 w-4 mr-1.5" /> Add Model
              </Button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Model</TableHead>
                  <TableHead>Provider</TableHead>
                  <TableHead>Cost (In/Out per 1K)</TableHead>
                  <TableHead>Tags</TableHead>
                  <TableHead>Priority</TableHead>
                  <TableHead>Active</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {models.map((m) => (
                  <TableRow key={m.id} className="group">
                    <TableCell>
                      <div>
                        <div className="font-medium text-sm">{m.display_name || m.provider_model}</div>
                        {m.display_name && (
                          <div className="text-xs text-muted-foreground font-mono">{m.provider_model}</div>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline" className="text-xs font-normal">
                        {m.provider}
                      </Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {formatCost(m.input_cost_per_1k)} / {formatCost(m.output_cost_per_1k)}
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {(m.tags || []).map((t) => (
                          <Badge key={t} variant="secondary" className="text-[10px] px-1.5 py-0">
                            {t}
                          </Badge>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell>
                      <span className="text-sm tabular-nums">{m.priority}</span>
                    </TableCell>
                    <TableCell>
                      <Switch
                        checked={m.is_active}
                        onCheckedChange={(v) => handleToggle(m.id, v)}
                      />
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                        <Button size="icon" variant="ghost" className="h-8 w-8" onClick={() => openEdit(m)}>
                          <Pencil className="h-3.5 w-3.5" />
                        </Button>
                        <Button
                          size="icon"
                          variant="ghost"
                          className="h-8 w-8 text-destructive hover:text-destructive"
                          onClick={() => handleDelete(m.id)}
                        >
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
    </div>
  );
}
