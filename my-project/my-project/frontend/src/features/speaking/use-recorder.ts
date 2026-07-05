// React binding for the Recorder state machine.
import { useCallback, useEffect, useRef, useState } from 'react';
import { Recorder, type Recording, type RecorderState } from './recorder';

export function useRecorder() {
  const [state, setState] = useState<RecorderState>('idle');
  const [recording, setRecording] = useState<Recording | null>(null);
  const [error, setError] = useState('');
  const recorderRef = useRef<Recorder | null>(null);

  useEffect(() => () => recorderRef.current?.cancel(), []);

  const start = useCallback(async () => {
    setRecording(null);
    setError('');
    const recorder = new Recorder({
      onState: setState,
      onFinish: setRecording,
      onError: setError,
    });
    recorderRef.current = recorder;
    await recorder.start();
  }, []);

  const stop = useCallback(() => recorderRef.current?.stop(), []);
  const reset = useCallback(() => {
    recorderRef.current?.cancel();
    setRecording(null);
    setState('idle');
  }, []);

  return { state, recording, error, start, stop, reset };
}
