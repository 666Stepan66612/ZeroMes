import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { hasStoredKey } from '@/lib/crypto';

export function HomePage() {
  const navigate = useNavigate();

  useEffect(() => {
    // Check if user has stored keys
    if (hasStoredKey()) {
      // User is logged in, redirect to chats
      navigate('/chats', { replace: true });
    } else {
      // User is not logged in, redirect to login
      navigate('/login', { replace: true });
    }
  }, [navigate]);

  return (
    <div style={{ 
      display: 'flex', 
      justifyContent: 'center', 
      alignItems: 'center', 
      height: '100vh' 
    }}>
      <p>Loading...</p>
    </div>
  );
}
