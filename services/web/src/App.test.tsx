import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import App from "./App";

const person = {
  id: 1,
  name: "Alex Morgan",
  email: "alex@example.com",
  created_at: "2026-01-01T12:00:00Z",
};
const response = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });

describe("team directory", () => {
  it("loads and filters people, then fetches person details", async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(response([person]))
      .mockResolvedValueOnce(response(person));
    vi.stubGlobal("fetch", fetcher);
    render(<App />);
    expect(await screen.findByText("alex@example.com")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Search people"), {
      target: { value: "missing" },
    });
    expect(screen.getByText("No matches yet.")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Search people"), {
      target: { value: "" },
    });
    fireEvent.click(screen.getByRole("button", { name: "View Alex Morgan" }));
    expect(
      await screen.findByRole("link", { name: "alex@example.com" }),
    ).toHaveAttribute("href", "mailto:alex@example.com");
    expect(fetcher).toHaveBeenLastCalledWith(
      "/api/v1/users/alex%40example.com",
      undefined,
    );
  });

  it("creates a person, adds the returned row, and reports duplicate emails", async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(response([]))
      .mockResolvedValueOnce(response(person, 201))
      .mockResolvedValueOnce(response({ error: "email already exists" }, 409));
    vi.stubGlobal("fetch", fetcher);
    render(<App />);
    await screen.findByText("Good teams start with one person.");
    const fill = () => {
      fireEvent.change(screen.getByLabelText("Full name"), {
        target: { value: person.name },
      });
      fireEvent.change(screen.getByLabelText("Email address"), {
        target: { value: person.email },
      });
      fireEvent.click(screen.getByRole("button", { name: "Add to team" }));
    };
    fill();
    expect(
      await screen.findByText("Alex Morgan was added to your team."),
    ).toBeInTheDocument();
    expect(screen.getByText(person.email)).toBeInTheDocument();
    expect(fetcher).toHaveBeenNthCalledWith(
      2,
      "/api/v1/users",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ name: person.name, email: person.email }),
      }),
    );
    fill();
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "This email is already in your team.",
    );
    expect(screen.getAllByText(person.email)).toHaveLength(1);
  });

  it("recovers after the list request fails", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockRejectedValueOnce(new TypeError("offline"))
        .mockResolvedValueOnce(response([])),
    );
    render(<App />);
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Could not connect.",
    );
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    await waitFor(() =>
      expect(
        screen.getByText("Good teams start with one person."),
      ).toBeInTheDocument(),
    );
  });
});
