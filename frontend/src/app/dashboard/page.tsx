"use client";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Activity, CheckCircle, Server, XCircle } from "lucide-react";
import { TaskList } from "./components/task-list";
import { useMetrics } from "@/hooks/use-metrics";
import { CreateTaskForm } from "./components/create-task-form";

export default function DashboardPage() {
  const { data: metrics, isLoading } = useMetrics();

  const MetricCardSkeleton = () => (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <div className="h-4 bg-muted rounded w-3/4"></div>
      </CardHeader>
      <CardContent>
        <div className="h-8 bg-muted rounded w-1/2"></div>
        <div className="h-3 bg-muted rounded w-1/4 mt-2"></div>
      </CardContent>
    </Card>
  );

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold tracking-tight text-foreground">
            Dashboard
          </h1>
          <p className="mt-1 text-muted-foreground">
            A real-time overview of your task scheduling system.
          </p>
        </div>
        <div>
          <CreateTaskForm />
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4 mb-8">
        {isLoading ? (
          <>
            <MetricCardSkeleton />
            <MetricCardSkeleton />
            <MetricCardSkeleton />
            <MetricCardSkeleton />
          </>
        ) : (
          <>
            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <CardTitle className="text-sm font-medium">
                  Total Tasks
                </CardTitle>
                <Server className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{metrics?.totalTasks}</div>
                <p className="text-xs text-muted-foreground">
                  +2 from last hour
                </p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <CardTitle className="text-sm font-medium">
                  Tasks Running
                </CardTitle>
                <Activity className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{metrics?.running}</div>
                <p className="text-xs text-muted-foreground">Live now</p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <CardTitle className="text-sm font-medium">
                  Completed Today
                </CardTitle>
                <CheckCircle className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {metrics?.completedToday}
                </div>
                <p className="text-xs text-muted-foreground">
                  +12% from yesterday
                </p>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-2">
                <CardTitle className="text-sm font-medium text-destructive">
                  Failed Tasks
                </CardTitle>
                <XCircle className="h-4 w-4 text-destructive" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-destructive">
                  {metrics?.failed}
                </div>
                <p className="text-xs text-muted-foreground">Needs attention</p>
              </CardContent>
            </Card>
          </>
        )}
      </div>

      <div>
        <TaskList />
      </div>
    </div>
  );
}
