import { useEffect, useRef, useState, type FormEvent } from "react";
import { createUser, getUser, listUsers, type User } from "./api";

const date = (value: string) =>
  new Intl.DateTimeFormat("en", { dateStyle: "medium" }).format(
    new Date(value),
  );
const initials = (name: string) =>
  name
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((word) => word[0])
    .join("")
    .toUpperCase();
const message = (error: unknown) =>
  error instanceof Error ? error.message : "Something went wrong.";

export default function App() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [filter, setFilter] = useState("");
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState("");
  const [notice, setNotice] = useState("");
  const [selected, setSelected] = useState<User | null>(null);
  const [detailError, setDetailError] = useState("");
  const [detailLoading, setDetailLoading] = useState(false);
  // Ignore an old list response if a mutation or newer load has superseded it.
  const revision = useRef(0);
  const detailRevision = useRef(0);

  async function refresh() {
    const current = ++revision.current;
    setLoading(true);
    setLoadError("");
    try {
      const result = await listUsers();
      if (revision.current === current) setUsers(result);
    } catch (error) {
      if (revision.current === current) setLoadError(message(error));
    } finally {
      if (revision.current === current) setLoading(false);
    }
  }

  useEffect(() => {
    void refresh();
    return () => {
      revision.current++;
      detailRevision.current++;
    };
  }, []);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    setSaving(true);
    setFormError("");
    setNotice("");
    try {
      const user = await createUser({
        name: String(data.get("name")).trim(),
        email: String(data.get("email")).trim(),
      });
      revision.current++;
      setUsers((previous) => [
        ...previous.filter((item) => item.id !== user.id),
        user,
      ]);
      setLoading(false);
      setLoadError("");
      setNotice(`${user.name} was added to your team.`);
      form.reset();
    } catch (error) {
      setFormError(message(error));
    } finally {
      setSaving(false);
    }
  }

  async function showDetails(email: string) {
    const current = ++detailRevision.current;
    setDetailLoading(true);
    setDetailError("");
    setSelected(null);
    try {
      const user = await getUser(email);
      if (detailRevision.current === current) setSelected(user);
    } catch (error) {
      if (detailRevision.current === current) setDetailError(message(error));
    } finally {
      if (detailRevision.current === current) setDetailLoading(false);
    }
  }

  const visible = users.filter((user) =>
    `${user.name} ${user.email}`.toLowerCase().includes(filter.toLowerCase()),
  );

  return (
    <div className="shell">
      <header className="topbar">
        <a className="brand" href="/" aria-label="Workspace home">
          <span className="brand-mark">w.</span> workspace
        </a>
        <span className="workspace-label">
          TEAM SPACE <span className="dot" />
        </span>
      </header>
      <main>
        <div className="breadcrumb">
          Workspace <span>/</span> People
        </div>
        <section className="page-heading">
          <div>
            <p className="eyebrow">BETTER TOGETHER</p>
            <h1>Your people.</h1>
            <p className="intro">
              A little directory for the team you're building.
            </p>
          </div>
          <div className="count">
            <strong>{users.length.toString().padStart(2, "0")}</strong>
            <span>TEAM {users.length === 1 ? "MEMBER" : "MEMBERS"}</span>
          </div>
        </section>
        <div className="layout">
          <section
            className="panel directory"
            aria-labelledby="directory-title"
          >
            <div className="section-heading">
              <h2 id="directory-title">
                Team directory <span className="badge">{users.length}</span>
              </h2>
              <button
                className="text-button"
                onClick={() => void refresh()}
                disabled={loading}
              >
                ↻ Refresh
              </button>
            </div>
            <label className="search">
              <span aria-hidden="true">⌕</span>
              <input
                aria-label="Search people"
                placeholder="Find a name or email…"
                value={filter}
                onChange={(event) => setFilter(event.target.value)}
              />
            </label>
            {loadError && (
              <div className="error" role="alert">
                {loadError}{" "}
                <button className="text-button" onClick={() => void refresh()}>
                  Try again
                </button>
              </div>
            )}
            {loading && (
              <p className="empty" role="status">
                Loading your team…
              </p>
            )}
            {!loading && !loadError && visible.length === 0 && (
              <div className="empty">
                <span className="empty-icon" aria-hidden="true">
                  ◎
                </span>
                <h3>
                  {users.length
                    ? "No matches yet."
                    : "Good teams start with one person."}
                </h3>
                <p>
                  {users.length
                    ? "Try another name or email address."
                    : "Add your first teammate using the form."}
                </p>
              </div>
            )}
            {!loading && visible.length > 0 && (
              <div className="table-scroll">
                <table>
                  <thead>
                    <tr>
                      <th scope="col">PERSON</th>
                      <th scope="col">JOINED</th>
                      <th scope="col">
                        <span className="sr-only">Details</span>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {visible.map((user) => (
                      <tr key={user.id}>
                        <td>
                          <div className="person">
                            <span className={`avatar tone-${user.id % 3}`}>
                              {initials(user.name)}
                            </span>
                            <div>
                              <strong>{user.name}</strong>
                              <span>{user.email}</span>
                            </div>
                          </div>
                        </td>
                        <td className="joined">{date(user.created_at)}</td>
                        <td>
                          <button
                            className="detail-button"
                            aria-label={`View ${user.name}`}
                            onClick={() => void showDetails(user.email)}
                          >
                            ↗
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            <div className="directory-footer">
              {loading
                ? "Connecting to your workspace"
                : `${visible.length} of ${users.length} people`}
              <span>Made for your team.</span>
            </div>
          </section>
          <aside>
            <section className="panel add-person" aria-labelledby="add-title">
              <span className="form-icon" aria-hidden="true">
                ＋
              </span>
              <h2 id="add-title">Add a teammate</h2>
              <p>Make room for someone new.</p>
              <form onSubmit={(event) => void submit(event)}>
                <label htmlFor="name">Full name</label>
                <input
                  id="name"
                  name="name"
                  autoComplete="name"
                  placeholder="e.g. Alex Morgan"
                  required
                  maxLength={200}
                  disabled={saving}
                />
                <label htmlFor="email">Email address</label>
                <input
                  id="email"
                  name="email"
                  type="email"
                  autoComplete="email"
                  placeholder="alex@company.com"
                  required
                  maxLength={254}
                  disabled={saving}
                />
                {formError && (
                  <p className="error" role="alert">
                    {formError}
                  </p>
                )}
                {notice && (
                  <p className="success" role="status">
                    {notice}
                  </p>
                )}
                <button className="primary" type="submit" disabled={saving}>
                  {saving ? "Adding…" : "Add to team"}
                  <span aria-hidden="true">↗</span>
                </button>
              </form>
              <small>
                This adds a directory entry. No invitation email is sent.
              </small>
            </section>
            <section className="note">
              <span aria-hidden="true">✳</span>
              <div>
                <h3>Small team. Big possibilities.</h3>
                <p>
                  Keep everyone's contact details in one place, from the very
                  first hello.
                </p>
              </div>
            </section>
            <section className="details" aria-live="polite">
              {detailLoading && <p>Loading person…</p>}
              {detailError && (
                <p role="alert" className="error">
                  {detailError}
                </p>
              )}
              {selected && (
                <>
                  <div className="section-heading">
                    <h2>Person details</h2>
                    <button
                      className="text-button"
                      onClick={() => {
                        detailRevision.current++;
                        setSelected(null);
                      }}
                    >
                      Close
                    </button>
                  </div>
                  <strong>{selected.name}</strong>
                  <a href={`mailto:${selected.email}`}>{selected.email}</a>
                  <p>Joined {date(selected.created_at)}</p>
                </>
              )}
            </section>
          </aside>
        </div>
        <footer className="page-footer">
          <span>WORKSPACE / PEOPLE</span>
          <span>A place for everyone.</span>
        </footer>
      </main>
    </div>
  );
}
