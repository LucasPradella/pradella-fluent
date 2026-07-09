// Shared pt-BR auth form pieces (US1 — FR-001).
import type { ReactNode } from 'react';

export function AuthLayout({ title, children }: { title: string; children: ReactNode }) {
  return (
    <main
      style={{
        maxWidth: 420,
        margin: '0 auto',
        padding: 24,
        display: 'flex',
        flexDirection: 'column',
        gap: 16,
        minHeight: '100%',
        justifyContent: 'center',
      }}
    >
      <h1 style={{ textAlign: 'center' }}>FluentDev</h1>
      <p style={{ textAlign: 'center', color: 'var(--text-secondary)', marginTop: -8 }}>
        Inglês comunicativo e técnico para devs
      </p>
      <h2>{title}</h2>
      {children}
    </main>
  );
}

export function OAuthButtons() {
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      <a
        href="/api/v1/auth/oauth/github/start"
        className="card"
        style={{ textAlign: 'center', textDecoration: 'none', color: 'var(--text-primary)' }}
      >
        Continuar com GitHub
      </a>
      <a
        href="/api/v1/auth/oauth/google/start"
        className="card"
        style={{ textAlign: 'center', textDecoration: 'none', color: 'var(--text-primary)' }}
      >
        Continuar com Google
      </a>
      <div
        aria-hidden
        style={{ textAlign: 'center', color: 'var(--text-disabled)', margin: '4px 0' }}
      >
        — ou —
      </div>
    </div>
  );
}
