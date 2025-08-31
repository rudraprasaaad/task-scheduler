export default function DashboardPage() {
  return (
    <div>
      <h2 className="text-3xl font-bold tracking-tight">Dashboard Overview</h2>
      <p className="mt-2 text-gray-600">
        Welcome! Here is a summary of your task scheduler system.
      </p>

      <div className="mt-8 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {/* Metric card of tasks will go here */}
      </div>
    </div>
  );
}
