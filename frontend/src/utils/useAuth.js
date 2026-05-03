const TOKEN_KEY = 'token';
const USER_KEY = 'user';

export const useAuth = () => {
  const getToken = () => localStorage.getItem(TOKEN_KEY);

  const getUser = () => {
    try {
      return JSON.parse(localStorage.getItem(USER_KEY) || '{}');
    } catch {
      return {};
    }
  };

  const saveSession = (token, user) => {
    localStorage.setItem(TOKEN_KEY, token);
    localStorage.setItem(USER_KEY, JSON.stringify(user));
  };

  const clearSession = () => {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(USER_KEY);
  };

  return {
    isAuthenticated: Boolean(getToken()),
    getUser,
    saveSession,
    clearSession,
  };
};
