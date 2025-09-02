export enum TaskStatus {
  PENDING = "pending",
  RUNNING = "running",
  COMPLETED = "completed",
  FAILED = "failed",
}

export enum TaskType {
  EMAIL = "email",
  NOTIFICATION = "notification",
  REPORT = "report",
  MAINTENANCE = "maintenance",
}

export interface Task {
  id: string;
  name: string;
  type: TaskType;
  status: TaskStatus;
  priority: number;
  retries: number;
  createdAt: string;
}
