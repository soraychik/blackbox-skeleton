const ROLE_LABELS = {
  admin: 'Администратор',
  engineer: 'Инженер',
  operator: 'Оператор',
};

export const useRole = () => {
  try {
    const user = JSON.parse(localStorage.getItem('user') || '{}');
    return {
      role: user.role || null,
      login: user.login || '',
      label: ROLE_LABELS[user.role] || user.role || '',
    };
  } catch {
    return { role: null, login: '', label: '' };
  }
};

export const canAccess = (role, ...allowed) => allowed.includes(role);
