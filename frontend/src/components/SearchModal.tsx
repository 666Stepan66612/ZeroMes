import { useState } from 'react';
import type { FormEvent } from 'react';
import { searchUsers } from '@/lib/api';
import type { User } from '@/types/api';

interface SearchModalProps {
  onClose: () => void;
  onSelectUser: (user: User) => void;
}

export function SearchModal({ onClose, onSelectUser }: SearchModalProps) {
  const [query, setQuery] = useState('');
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);

  const handleSearch = async (e: FormEvent) => {
    e.preventDefault();

    if (!query.trim()) return;

    try {
      setLoading(true);
      setSearched(true);
      const results = await searchUsers(query);
      // Ensure results is an array
      setUsers(Array.isArray(results) ? results : []);
    } catch (error) {
      console.error('Search error:', error);
      setUsers([]);
    } finally {
      setLoading(false);
    }
  };

  const handleSelectUser = (user: User) => {
    onSelectUser(user);
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h3>New Chat</h3>
          <button className="btn-close" onClick={onClose}>
            ✕
          </button>
        </div>

        <form className="search-form" onSubmit={handleSearch}>
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search by login..."
            autoFocus
          />
          <button type="submit" className="btn-primary" disabled={loading}>
            {loading ? 'Searching...' : 'Search'}
          </button>
        </form>

        <div className="search-results">
          {loading ? (
            <div className="loading">Searching...</div>
          ) : searched && users.length === 0 ? (
            <div className="no-results">
              <p>No users found</p>
              <p className="help-text">Try a different search term</p>
            </div>
          ) : (
            users.map((user) => (
              <div
                key={user.id}
                className="user-item"
                onClick={() => handleSelectUser(user)}
              >
                <div className="user-avatar">
                  {user.login.substring(0, 2).toUpperCase()}
                </div>
                <div className="user-info">
                  <span className="user-login">{user.login}</span>
                  <span className="user-id">{user.id}</span>
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
