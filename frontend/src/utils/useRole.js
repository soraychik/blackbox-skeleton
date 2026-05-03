import { useAuth } from './useAuth';

const ROLE_LABELS = {
  admin: 'Администратор',
  engineer: 'Инженер',
  operator: 'Оператор',
};

export const useRole = () => {
  const { getUser } = useAuth();
  const user = getUser();
  return {
    role: user.role || null,
    login: user.login || '',
    label: ROLE_LABELS[user.role] || user.role || '',
  };
};

export const canAccess = (role, ...allowed) => allowed.includes(role);
