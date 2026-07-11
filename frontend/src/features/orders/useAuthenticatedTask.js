import { useState } from 'react';
import { ApiError } from '../../api/client';
import { clearToken, isWebMode } from '../../lib/auth';
import { logWarn } from '../../lib/logger';

// Shared async lifecycle for the orders orchestrator. Authentication expiry is
// handled consistently for every query and mutation executed through run().
export function useAuthenticatedTask(onAuthenticationRequired) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  async function run(task) {
    setLoading(true);
    setError('');
    try {
      return await task();
    } catch (err) {
      if (err instanceof ApiError && err.status === 401 && isWebMode()) {
        clearToken();
        onAuthenticationRequired();
        return undefined;
      }
      logWarn('action.failed', { error: err instanceof Error ? err.message : String(err) });
      setError(err instanceof Error ? err.message : String(err));
      return undefined;
    } finally {
      setLoading(false);
    }
  }

  return { error, loading, run, setError };
}
