// Offline awareness banner (US5 scenario 3) — pt-BR copy.
import { useOnline } from '../offline/connectivity';

export function OfflineBanner() {
  const online = useOnline();
  if (online) return null;
  return (
    <div
      role="status"
      style={{
        background: 'var(--surface-2)',
        color: 'var(--warning)',
        padding: '8px 16px',
        textAlign: 'center',
        fontSize: 14,
      }}
    >
      Você está offline. Seu progresso será sincronizado quando a conexão voltar.
    </div>
  );
}
