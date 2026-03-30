import { allSections, scopes } from "../constants";
import { AuthState, NavSection } from "../types";

type ContextBarProps = {
  authStatus: AuthState;
  activeScope: string;
  handleRefresh: () => void;
  setActiveScope: (scope: string) => void;
  setActiveSection: (section: NavSection) => void;
};

export function ContextBar({
  authStatus,
  activeScope,
  handleRefresh,
  setActiveScope,
  setActiveSection,
}: ContextBarProps) {
  const authConfigured = authStatus.authenticated;

  return (
    <header className="context-bar">
      <div className="context-app">
        {authConfigured ? (
          <>
            <span className="context-badge">{authStatus.storage || "Authenticated"}</span>
            {authStatus.profile && (
              <span className="context-version">{authStatus.profile}</span>
            )}
            <span className="context-status state-ready">Connected</span>
          </>
        ) : (
          <span className="context-status state-processing">Not authenticated</span>
        )}
      </div>
      <div className="toolbar-right">
        <div className="scope-tabs" role="tablist" aria-label="Scope">
          {scopes.map((scope) => (
            <button
              key={scope.id}
              type="button"
              role="tab"
              aria-selected={activeScope === scope.id}
              className={`scope-tab ${activeScope === scope.id ? "is-active" : ""}`}
              onClick={() => {
                setActiveScope(scope.id);
                const firstSection = scope.groups[0]?.items[0];
                if (firstSection) setActiveSection(firstSection);
              }}
            >
              {scope.label}
            </button>
          ))}
        </div>
        <button
          className="toolbar-btn"
          type="button"
          onClick={handleRefresh}
          aria-label="Refresh (⌘R)"
          title="Refresh (⌘R)"
        >
          <span aria-hidden="true">↻</span>
        </button>
        {!authConfigured && (
          <button
            className="toolbar-btn"
            type="button"
            onClick={() => setActiveSection(allSections.find((s) => s.id === "settings")!)}
          >
            Configure
          </button>
        )}
      </div>
    </header>
  );
}
