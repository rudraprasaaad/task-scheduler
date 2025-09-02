"use client";

import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { dummyTasks } from "@/lib/dummy-data";
import { Task, TaskStatus } from "@/lib/type";
import { useQuery } from "@tanstack/react-query";

const fetchTasks = async (): Promise<Task[]> => {
  return new Promise((resolve) => {
    setTimeout(() => {
      resolve(dummyTasks);
    }, 1000);
  });
};

const getStatusBadgeVariant = (status: TaskStatus) => {
  switch (status) {
    case TaskStatus.COMPLETED:
      return "default";
    case TaskStatus.RUNNING:
      return "secondary";
    case TaskStatus.PENDING:
      return "outline";
    case TaskStatus.FAILED:
      return "destructive";
    default:
      return "default";
  }
};

export function TaskList() {
  const {
    data: tasks,
    isLoading,
    isError,
    error,
  } = useQuery<Task[], Error>({
    queryKey: ["tasks"],
    queryFn: fetchTasks,
  });

  if (isLoading) return <div>Loading tasks...</div>;

  if (isError) return <div>Error fetching tasks: {error.message}</div>;

  return (
    <div className="mt-8">
      <h3 className="text-xl font-semibold">Recent Tasks</h3>
      <div className="mt-4 rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Task Name</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Priority</TableHead>
              <TableHead>Created At</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {tasks?.map((task) => (
              <TableRow key={task.id}>
                <TableCell className="font-medium">{task.name}</TableCell>
                <TableCell>{task.type}</TableCell>
                <TableCell>
                  <Badge variant={getStatusBadgeVariant(task.status)}>
                    {task.status}
                  </Badge>
                </TableCell>
                <TableCell>{task.priority}</TableCell>
                <TableCell>
                  {new Date(task.createdAt).toLocaleString()}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
