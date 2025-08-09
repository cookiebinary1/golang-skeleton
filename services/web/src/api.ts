export interface User {
  id: number;
  name: string;
  email: string;
  created_at: string;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`/api/v1${path}`, init);
  } catch {
    throw new Error("Could not connect. Please try again.");
  }
  if (!response.ok) {
    if (response.status === 409)
      throw new Error("This email is already in your team.");
    if (response.status === 400)
      throw new Error("Enter a name and a valid email address.");
    if (response.status === 404)
      throw new Error("This person could not be found.");
    throw new Error("Something went wrong. Please try again.");
  }
  return response.json() as Promise<T>;
}

export const listUsers = () => request<User[]>("/users");
export const getUser = (email: string) =>
  request<User>(`/users/${encodeURIComponent(email)}`);
export const createUser = (user: Pick<User, "name" | "email">) =>
  request<User>("/users", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(user),
  });
