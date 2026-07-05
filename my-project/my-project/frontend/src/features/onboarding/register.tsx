import { type FormEvent, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { api, ApiError, OfflineError } from '../../shared/api/client';
import type { User } from '../../shared/api/types';
import { AuthLayout, OAuthButtons } from './auth-form';

export function RegisterPage() {
  const [displayName, setDisplayName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    if (password.length < 10) {
      setError('A senha deve ter pelo menos 10 caracteres.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      const user = await api.post<User>('/auth/register', { email, password, displayName });
      queryClient.setQueryData(['me'], user);
      // New account goes straight to the placement test (FR-002).
      navigate('/placement');
    } catch (err) {
      if (err instanceof OfflineError) {
        setError('Criar conta requer conexão com a internet.');
      } else if (err instanceof ApiError && err.status === 409) {
        setError('Este e-mail já está cadastrado. Tente entrar.');
      } else if (err instanceof ApiError && err.status === 400) {
        setError(err.problem.detail || 'Dados inválidos. Confira e tente de novo.');
      } else {
        setError('Não foi possível criar a conta. Tente novamente.');
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <AuthLayout title="Criar conta">
      <OAuthButtons />
      <form onSubmit={onSubmit} style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <label>
          Nome
          <input
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            required
            maxLength={60}
            autoComplete="name"
          />
        </label>
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
          Senha (mínimo 10 caracteres)
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            minLength={10}
            maxLength={128}
            autoComplete="new-password"
          />
        </label>
        {error && (
          <p role="alert" style={{ color: 'var(--error)' }}>
            {error}
          </p>
        )}
        <button type="submit" disabled={busy}>
          {busy ? 'Criando…' : 'Criar conta e fazer o teste de nível'}
        </button>
      </form>
      <p>
        Já tem conta? <Link to="/login">Entrar</Link>
      </p>
    </AuthLayout>
  );
}
