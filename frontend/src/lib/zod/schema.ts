import { z } from "zod";
import { TaskType } from "../type";

export const formSchema = z.object({
  name: z.string().min(3, "Task name must be at least 3 characters."),
  type: z.enum([
    TaskType.EMAIL,
    TaskType.NOTIFICATION,
    TaskType.REPORT,
    TaskType.MAINTENANCE,
  ]),
  priority: z.number().min(1).max(10),
});
