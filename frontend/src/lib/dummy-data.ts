import { Task, TaskStatus, TaskType } from "./type";

export const dummyTasks: Task[] = [
  {
    id: "ts_1",
    name: "Send Welcome Email",
    type: TaskType.EMAIL,
    status: TaskStatus.COMPLETED,
    priority: 5,
    retries: 0,
    createdAt: "2025-09-01T10:00:00Z",
  },
  {
    id: "ts_2",
    name: "Generate Daily Report",
    type: TaskType.REPORT,
    status: TaskStatus.RUNNING,
    priority: 8,
    retries: 1,
    createdAt: "2025-09-01T11:00:00Z",
  },
  {
    id: "ts_3",
    name: "Push Notification Campaign",
    type: TaskType.NOTIFICATION,
    status: TaskStatus.PENDING,
    priority: 3,
    retries: 0,
    createdAt: "2025-09-01T11:30:00Z",
  },
  {
    id: "ts_4",
    name: "Database Maintenance",
    type: TaskType.MAINTENANCE,
    status: TaskStatus.FAILED,
    priority: 10,
    retries: 3,
    createdAt: "2025-09-01T09:00:00Z",
  },
  {
    id: "ts_5",
    name: "Send Password Reset",
    type: TaskType.EMAIL,
    status: TaskStatus.COMPLETED,
    priority: 9,
    retries: 0,
    createdAt: "2025-09-01T11:45:00Z",
  },
];
