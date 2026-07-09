// Install hint (US5): captures beforeinstallprompt on Android/Chrome and
// shows manual instructions on iOS Safari (which has no prompt API).
import { useEffect, useState } from 'react';

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
}

function isIOSSafari(): boolean {
  const ua = navigator.userAgent;
  return /iPhone|iPad|iPod/.test(ua) && /Safari/.test(ua) && !/CriOS|FxiOS/.test(ua);
}

function isStandalone(): boolean {
  return (
    window.matchMedia('(display-mode: standalone)').matches ||
    ('standalone' in navigator && (navigator as { standalone?: boolean }).standalone === true)
  );
}

export function InstallHint() {
  const [promptEvent, setPromptEvent] = useState<BeforeInstallPromptEvent | null>(null);
  const [dismissed, setDismissed] = useState(() => localStorage.getItem('install-hint') === 'off');

  useEffect(() => {
    const handler = (e: Event) => {
      e.preventDefault();
      setPromptEvent(e as BeforeInstallPromptEvent);
    };
    window.addEventListener('beforeinstallprompt', handler);
    return () => window.removeEventListener('beforeinstallprompt', handler);
  }, []);

  if (dismissed || isStandalone()) return null;

  const dismiss = () => {
    localStorage.setItem('install-hint', 'off');
    setDismissed(true);
  };

  if (promptEvent) {
    return (
      <div className="card" style={{ margin: 16, display: 'flex', gap: 12, alignItems: 'center' }}>
        <span style={{ flex: 1 }}>Instale o FluentDev para estudar em tela cheia, até offline.</span>
        <button onClick={() => void promptEvent.prompt()}>Instalar</button>
        <button className="secondary" onClick={dismiss} aria-label="Dispensar">
          ✕
        </button>
      </div>
    );
  }

  if (isIOSSafari()) {
    return (
      <div className="card" style={{ margin: 16, display: 'flex', gap: 12, alignItems: 'center' }}>
        <span style={{ flex: 1 }}>
          Para instalar no iPhone: toque em <strong>Compartilhar</strong> e depois em{' '}
          <strong>Adicionar à Tela de Início</strong>.
        </span>
        <button className="secondary" onClick={dismiss} aria-label="Dispensar">
          ✕
        </button>
      </div>
    );
  }
  return null;
}
