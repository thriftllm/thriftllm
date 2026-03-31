"use client";

import { useAuth } from "@/lib/auth";
import { useRouter, usePathname } from "next/navigation";
import { useEffect } from "react";
import Link from "next/link";
import {
  LayoutDashboard,
  Bot,
  Key,
  FileText,
  BarChart3,
  Database,
  Settings,
  LogOut,
  Zap,
  GitBranch,
} from "lucide-react";
import { cn } from "@/lib/utils";

const platformNav = [
  { label: "Overview", href: "/overview", icon: LayoutDashboard },
  { label: "Models", href: "/models", icon: Bot },
  { label: "Chains", href: "/chains", icon: GitBranch },
  { label: "API Keys", href: "/api-keys", icon: Key },
];

const dataNav = [
  { label: "Requests", href: "/requests", icon: FileText },
  { label: "Analytics", href: "/analytics", icon: BarChart3 },
  { label: "Cache", href: "/cache", icon: Database },
];

const allNav = [
  ...platformNav,
  ...dataNav,
  { label: "Settings", href: "/settings", icon: Settings },
];

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { user, loading, logout } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (!loading && !user) router.replace("/login");
  }, [loading, user, router]);

  if (loading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-muted border-t-foreground" />
      </div>
    );
  }

  if (!user) return null;

  const isActive = (href: string) =>
    href === "/overview" ? pathname === "/overview" : pathname.startsWith(href);

  const currentPage = allNav.find((n) => isActive(n.href));

  return (
    <div className="flex h-screen overflow-hidden">
      {/* Sidebar */}
      <aside className="flex w-56 flex-col border-r bg-sidebar-background">
        {/* Brand */}
        <div className="flex h-14 shrink-0 items-center gap-2 border-b border-sidebar-border px-4">
          <div className="flex h-6 w-6 items-center justify-center rounded-md bg-sidebar-primary">
            <Zap className="h-3.5 w-3.5 text-sidebar-primary-foreground" />
          </div>
          <span className="text-sm font-semibold tracking-tight">ThriftLLM</span>
        </div>

        {/* Navigation */}
        <nav className="flex-1 overflow-y-auto px-3 py-4 space-y-6">
          <NavSection label="Platform" items={platformNav} isActive={isActive} />
          <NavSection label="Data" items={dataNav} isActive={isActive} />
          <NavSection
            label="Settings"
            items={[{ label: "Account", href: "/settings", icon: Settings }]}
            isActive={isActive}
          />
        </nav>

        {/* User footer */}
        <div className="shrink-0 border-t border-sidebar-border p-3 space-y-1">
          <div className="flex items-center gap-2 rounded-md px-2 py-1.5">
            <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-muted text-[10px] font-semibold">
              {user.name?.charAt(0).toUpperCase()}
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium leading-none">{user.name}</p>
              <p className="truncate text-[11px] text-muted-foreground">{user.email}</p>
            </div>
          </div>
          <button
            onClick={() => { logout(); router.replace("/login"); }}
            className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground cursor-pointer"
          >
            <LogOut className="h-4 w-4" />
            Sign out
          </button>
        </div>
      </aside>

      {/* Main area */}
      <div className="flex flex-1 flex-col min-w-0">
        <header className="flex h-14 shrink-0 items-center border-b px-6">
          <div className="flex items-center gap-2 text-sm">
            <span className="text-muted-foreground">Dashboard</span>
            {currentPage && currentPage.href !== "/overview" && (
              <>
                <span className="text-muted-foreground/40">/</span>
                <span className="font-medium">{currentPage.label}</span>
              </>
            )}
            {currentPage?.href === "/overview" && (
              <>
                <span className="text-muted-foreground/40">/</span>
                <span className="font-medium">Overview</span>
              </>
            )}
          </div>
        </header>
        <main className="flex-1 overflow-auto">
          <div className="mx-auto max-w-screen-xl p-6">{children}</div>
        </main>
      </div>
    </div>
  );
}

function NavSection({
  label,
  items,
  isActive,
}: {
  label: string;
  items: { label: string; href: string; icon: React.ComponentType<{ className?: string }> }[];
  isActive: (href: string) => boolean;
}) {
  return (
    <div>
      <p className="mb-1.5 px-2 text-[11px] font-semibold uppercase tracking-wider text-sidebar-muted">
        {label}
      </p>
      <div className="space-y-0.5">
        {items.map((item) => (
          <Link
            key={item.href}
            href={item.href}
            className={cn(
              "flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors",
              isActive(item.href)
                ? "bg-sidebar-accent font-medium text-sidebar-accent-foreground"
                : "text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
            )}
          >
            <item.icon className="h-4 w-4" />
            {item.label}
          </Link>
        ))}
      </div>
    </div>
  );
}
