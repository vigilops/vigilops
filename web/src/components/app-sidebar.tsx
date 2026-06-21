import { Link, useRouterState } from "@tanstack/react-router"
import { Bot, LayoutDashboard, ListTree, Wrench } from "lucide-react"

import { Brand } from "@/components/brand"

import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"

const NAV = [
  { title: "Overview", to: "/dashboard", icon: LayoutDashboard },
  { title: "Agents", to: "/dashboard/agents", icon: Bot },
  { title: "Agent runs", to: "/dashboard/runs", icon: ListTree },
  { title: "Tool analytics", to: "/dashboard/tools", icon: Wrench },
] as const

export function AppSidebar() {
  const pathname = useRouterState({ select: (s) => s.location.pathname })

  return (
    <Sidebar>
      <SidebarHeader className="h-14 justify-center border-b px-4">
        <Brand />
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Observability</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {NAV.map((item) => (
                <SidebarMenuItem key={item.to}>
                  <SidebarMenuButton
                    isActive={item.to === "/dashboard" ? pathname === item.to : pathname.startsWith(item.to)}
                    tooltip={item.title}
                    render={<Link to={item.to} />}
                  >
                    <item.icon />
                    <span>{item.title}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  )
}
