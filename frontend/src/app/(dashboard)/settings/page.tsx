"use client";

import { useState } from "react";
import { useAuth } from "@/lib/auth";
import { updateProfile, changePassword } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Check, Copy, CheckCheck } from "lucide-react";

export default function SettingsPage() {
  const { user, updateUser } = useAuth();

  const [name, setName] = useState(user?.name || "");
  const [email, setEmail] = useState(user?.email || "");
  const [profileSaving, setProfileSaving] = useState(false);
  const [profileSuccess, setProfileSuccess] = useState(false);

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [passwordSaving, setPasswordSaving] = useState(false);
  const [passwordSuccess, setPasswordSuccess] = useState(false);
  const [passwordError, setPasswordError] = useState("");

  const [copied, setCopied] = useState(false);

  const handleProfileSave = async () => {
    setProfileSaving(true);
    setProfileSuccess(false);
    try {
      await updateProfile({ name, email });
      updateUser({ name, email });
      setProfileSuccess(true);
      setTimeout(() => setProfileSuccess(false), 3000);
    } catch (err: any) {
      alert(err.message);
    } finally {
      setProfileSaving(false);
    }
  };

  const handlePasswordChange = async () => {
    setPasswordError("");
    if (newPassword.length < 8) {
      setPasswordError("Password must be at least 8 characters");
      return;
    }
    if (newPassword !== confirmPassword) {
      setPasswordError("Passwords do not match");
      return;
    }
    setPasswordSaving(true);
    setPasswordSuccess(false);
    try {
      await changePassword({ current_password: currentPassword, new_password: newPassword });
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      setPasswordSuccess(true);
      setTimeout(() => setPasswordSuccess(false), 3000);
    } catch (err: any) {
      setPasswordError(err.message);
    } finally {
      setPasswordSaving(false);
    }
  };

  const curlExample = `curl http://localhost:8080/v1/chat/completions \\
  -H "Authorization: Bearer thr_your_key" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"Hello"}]}'`;

  const handleCopy = () => {
    navigator.clipboard.writeText(curlExample);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h2 className="text-lg font-semibold tracking-tight">Settings</h2>
        <p className="text-sm text-muted-foreground">Manage your account and view proxy configuration.</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Profile</CardTitle>
          <CardDescription>Update your display name and email address.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-2">
            <Label>Name</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="grid gap-2">
            <Label>Email</Label>
            <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          </div>
          <div className="flex items-center gap-3">
            <Button size="sm" onClick={handleProfileSave} disabled={profileSaving}>
              {profileSaving ? "Saving..." : "Save Changes"}
            </Button>
            {profileSuccess && (
              <span className="flex items-center gap-1 text-xs text-muted-foreground">
                <Check className="h-3.5 w-3.5" /> Saved
              </span>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Password</CardTitle>
          <CardDescription>Change your account password.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-2">
            <Label>Current Password</Label>
            <Input type="password" value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} />
          </div>
          <div className="grid sm:grid-cols-2 gap-4">
            <div className="grid gap-2">
              <Label>New Password</Label>
              <Input type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
            </div>
            <div className="grid gap-2">
              <Label>Confirm Password</Label>
              <Input type="password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} />
            </div>
          </div>
          {passwordError && (
            <div className="rounded-md border border-destructive/20 bg-destructive/5 px-3 py-2 text-xs text-destructive">
              {passwordError}
            </div>
          )}
          <div className="flex items-center gap-3">
            <Button size="sm" onClick={handlePasswordChange} disabled={passwordSaving || !currentPassword || !newPassword}>
              {passwordSaving ? "Changing..." : "Change Password"}
            </Button>
            {passwordSuccess && (
              <span className="flex items-center gap-1 text-xs text-muted-foreground">
                <Check className="h-3.5 w-3.5" /> Password updated
              </span>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Proxy Endpoint</CardTitle>
          <CardDescription>Drop-in replacement for OpenAI API. Compatible with any OpenAI SDK.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="rounded-md border p-3">
            <p className="text-[10px] uppercase tracking-wider text-muted-foreground font-medium mb-1.5">Base URL</p>
            <code className="text-sm font-mono font-medium">http://localhost:8080/v1</code>
          </div>
          <div className="rounded-md border p-3 relative group">
            <div className="flex items-center justify-between mb-1.5">
              <p className="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">Example</p>
              <Button variant="ghost" size="sm" className="h-6 px-2 text-xs opacity-0 group-hover:opacity-100 transition-opacity" onClick={handleCopy}>
                {copied ? <CheckCheck className="h-3 w-3 mr-1" /> : <Copy className="h-3 w-3 mr-1" />}
                {copied ? "Copied" : "Copy"}
              </Button>
            </div>
            <pre className="text-xs font-mono text-muted-foreground leading-relaxed whitespace-pre-wrap break-all">{curlExample}</pre>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
