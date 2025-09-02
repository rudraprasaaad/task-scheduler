import { dummyTasks } from "../dummy-data";
import { TaskStatus } from "../type";

export interface DashboardMetrics {
  totalTasks: number;
  running: number;
  completedToday: number;
  failed: number;
}

export const getDashboardMetrics = async (): Promise<DashboardMetrics> => {
  return new Promise((resolve) => {
    setTimeout(() => {
      resolve({
        totalTasks: dummyTasks.length,
        running: dummyTasks.filter((t) => t.status === TaskStatus.RUNNING)
          .length,
        completedToday: dummyTasks.filter(
          (t) => t.status === TaskStatus.COMPLETED
        ).length,
        failed: dummyTasks.filter((t) => t.status === TaskStatus.FAILED).length,
      });
    }, 500);
  });
};
