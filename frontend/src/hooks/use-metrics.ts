"use client";

import { getDashboardMetrics } from "@/lib/api/metrics";
import { useQuery } from "@tanstack/react-query";

export const useMetrics = () => {
  return useQuery({
    queryKey: ["dashboardMetrics"],
    queryFn: getDashboardMetrics,
    refetchInterval: 5000,
  });
};
