// @ts-check
import { api, unwrap } from './client';

// login exchanges username + password for a bearer session token.
// The auth middleware adds no header while there is no session yet.
/**
 * @param {string} username
 * @param {string} password
 */
export function login(username, password) {
  return unwrap(api.POST('/login', { body: { username, password } }));
}
