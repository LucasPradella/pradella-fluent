// Speaking exercise (US3): record ≤30 s, upload, show similarity + missed
// words. Permission denied or connectivity loss never blocks the lesson
// (FR-013..FR-016).
import { useState } from 'react';
import { api, ApiError, OfflineError } from '../../shared/api/client';
import type { Exercise, SpeechResult } from '../../shared/api/types';
import { useOnline } from '../../shared/offline/connectivity';
import { useRecorder } from './use-recorder';
import { SpeakingResult } from './result';

interface Props {
  exercise: Exercise;
  onFinished: (result: SpeechResult | null) => void; // null = skipped
}

export function SpeakingExercise({ exercise, onFinished }: Props) {
  const online = useOnline();
  const { state, recording, error, start, stop, reset } = useRecorder();
  const [uploading, setUploading] = useState(false);
  const [result, setResult] = useState<SpeechResult | null>(null);
  const [uploadError, setUploadError] = useState('');

  const target = exercise.targetSentence ?? '';

  async function upload() {
    if (!recording) return;
    setUploading(true);
    setUploadError('');
    try {
      const form = new FormData();
      form.set('attemptId', crypto.randomUUID());
      const ext = recording.mimeType === 'audio/mp4' ? 'mp4' : 'webm';
      form.set('audio', recording.blob, `attempt.${ext}`);
      const res = await api.post<SpeechResult>(
        `/exercises/${exercise.id}/speech-attempts`,
        form,
      );
      setResult(res);
    } catch (err) {
      if (err instanceof OfflineError) {
        setUploadError('A prática de fala precisa de conexão com a internet.');
      } else if (err instanceof ApiError && err.status === 422) {
        setUploadError('Não conseguimos entender o áudio. Tente gravar num lugar mais silencioso.');
      } else if (err instanceof ApiError && err.status === 503) {
        setUploadError('Avaliação de fala temporariamente indisponível. Tente de novo em instantes.');
      } else if (err instanceof ApiError && err.status === 429) {
        setUploadError('Muitas tentativas seguidas. Respire fundo e tente de novo em um minuto.');
      } else {
        setUploadError('Falha ao enviar o áudio. Tente novamente.');
      }
    } finally {
      setUploading(false);
    }
  }

  if (result) {
    return (
      <SpeakingResult
        target={target}
        result={result}
        onRetry={() => {
          setResult(null);
          reset();
        }}
        onContinue={() => onFinished(result)}
      />
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <p style={{ fontSize: 18 }}>{exercise.promptContext}</p>
      <blockquote
        className="card"
        style={{ margin: 0, fontSize: 20, fontWeight: 600, background: 'var(--surface-2)' }}
      >
        “{target}”
      </blockquote>

      {!online && (
        <div className="card" role="alert">
          <p style={{ color: 'var(--warning)' }}>
            📴 A prática de fala precisa de conexão. Você pode pular este exercício e voltar depois.
          </p>
          <button className="secondary" onClick={() => onFinished(null)}>
            Pular exercício
          </button>
        </div>
      )}

      {online && state === 'denied' && (
        <div className="card" role="alert">
          <p>
            Sem acesso ao microfone. Para praticar a fala, permita o microfone nas configurações do
            navegador — ou pule este exercício sem prejuízo na lição.
          </p>
          <div style={{ display: 'flex', gap: 8 }}>
            <button className="secondary" onClick={() => void start()}>
              Tentar de novo
            </button>
            <button className="secondary" onClick={() => onFinished(null)}>
              Pular exercício
            </button>
          </div>
        </div>
      )}

      {online && state !== 'denied' && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {state === 'idle' && (
            <button onClick={() => void start()} aria-label="Começar a gravar">
              🎙️ Gravar (máx. 30 s)
            </button>
          )}
          {state === 'requesting' && <button disabled>Aguardando permissão do microfone…</button>}
          {state === 'recording' && (
            <button onClick={stop} aria-label="Parar gravação" style={{ background: 'var(--error)' }}>
              ⏹️ Gravando… tocar para parar
            </button>
          )}
          {state === 'stopped' && recording && (
            <>
              <audio controls src={URL.createObjectURL(recording.blob)} aria-label="Sua gravação" />
              <div style={{ display: 'flex', gap: 8 }}>
                <button className="secondary" onClick={() => void start()} disabled={uploading}>
                  Regravar
                </button>
                <button onClick={() => void upload()} disabled={uploading} style={{ flex: 1 }}>
                  {uploading ? 'Avaliando…' : 'Enviar para avaliação'}
                </button>
              </div>
            </>
          )}
          {state === 'error' && (
            <div className="card" role="alert">
              <p style={{ color: 'var(--error)' }}>{error}</p>
              <button className="secondary" onClick={() => onFinished(null)}>
                Pular exercício
              </button>
            </div>
          )}
          {uploadError && (
            <div role="alert" className="card">
              <p style={{ color: 'var(--error)' }}>{uploadError}</p>
              <button className="secondary" onClick={() => onFinished(null)}>
                Pular exercício
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
