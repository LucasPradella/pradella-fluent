// Layout shell: top bar with streak, bottom navigation, offline banner.
import { Link, Outlet, useLocation } from 'react-router-dom';
import { useMe } from '../shared/api/hooks';
import { OfflineBanner } from '../shared/ui/offline-banner';
import { InstallHint } from '../pwa/install-hint';

export function Shell() {
  const { data: me } = useMe();
  const location = useLocation();

  const nav = [
    { to: '/dashboard', label: 'Painel', icon: '📊' },
    { to: '/tracks', label: 'Trilhas', icon: '🗺️' },
    { to: '/review', label: 'Revisão', icon: '🔁' },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', minHeight: '100%' }}>
      <OfflineBanner />
      <header
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '12px 16px',
          background: 'var(--surface-1)',
        }}
      >
        <Link to="/dashboard" style={{ textDecoration: 'none', fontWeight: 700, fontSize: 18 }}>
          FluentDev
        </Link>
        {me && (
          <span aria-label={`Sequência atual: ${me.currentStreak} dias`} style={{ fontWeight: 600 }}>
            🔥 {me.currentStreak}
          </span>
        )}
      </header>

      <main style={{ flex: 1, maxWidth: 720, width: '100%', margin: '0 auto', padding: 16 }}>
        <Outlet />
      </main>

      <InstallHint />

      <nav
        aria-label="Navegação principal"
        style={{
          display: 'flex',
          justifyContent: 'space-around',
          background: 'var(--surface-1)',
          padding: '8px 0',
          position: 'sticky',
          bottom: 0,
        }}
      >
        {nav.map((item) => {
          const active = location.pathname.startsWith(item.to);
          return (
            <Link
              key={item.to}
              to={item.to}
              aria-current={active ? 'page' : undefined}
              style={{
                textDecoration: 'none',
                color: active ? 'var(--accent)' : 'var(--text-secondary)',
                textAlign: 'center',
                fontSize: 13,
                minWidth: 80,
              }}
            >
              <div aria-hidden style={{ fontSize: 20 }}>{item.icon}</div>
              {item.label}
            </Link>
          );
        })}
      </nav>
    </div>
  );
}
