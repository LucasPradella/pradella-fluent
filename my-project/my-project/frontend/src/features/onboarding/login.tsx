import { type FormEvent, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { api, ApiError, OfflineError } from '../../shared/api/client';
import type { User } from '../../shared/api/types';
import { AuthLayout, OAuthButtons } from './auth-form';

export function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError('');
    try {
      const user = await api.post<User>('/auth/login', { email, password });
      queryClient.setQueryData(['me'], user);
      navigate(user.proficiencyLevel ? '/dashboard' : '/placement');
    } catch (err) {
      if (err instanceof OfflineError) {
        setError('Entrar requer conexão com a internet.');
      } else if (err instanceof ApiError && err.status === 401) {
        setError('E-mail ou senha incorretos.');
      } else if (err instanceof ApiError && err.status === 429) {
        setError('Muitas tentativas. Aguarde um instante e tente de novo.');
      } else {
        setError('Não foi possível entrar. Tente novamente.');
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <AuthLayout title="Entrar">
      <OAuthButtons />
      <form onSubmit={onSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <label>
          E-mail
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            autoComplete="email"
          />
        </label>
        <label>
          Senha
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoComplete="current-password"
          />
        </label>
        {error && (
          <p role="alert" style={{ color: 'var(--error)' }}>
            {error}
          </p>
        )}
        <button type="submit" disabled={busy}>
          {busy ? 'Entrando…' : 'Entrar'}
        </button>
      </form>
      <p>
        Ainda não tem conta? <Link to="/register">Criar conta</Link>
      </p>
    </AuthLayout>
  );
}
