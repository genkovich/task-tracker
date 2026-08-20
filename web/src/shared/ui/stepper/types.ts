export interface StepConfig {
  id: string;
  title: string;
  description?: string;
}

export type StepStatus = "completed" | "active" | "upcoming";
