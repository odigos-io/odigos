import React, { useRef, useState } from 'react';
import { SearchIcon } from '@/assets';
import { RelativeContainer } from '../styled';
import { Input } from '@/reuseable-components';
import { SearchResults } from './search-results';
import { useKeyDown, useOnClickOutside } from '@/hooks';
// import { RecentSearches } from './recent-searches';

const Search = () => {
  const [input, setInput] = useState('');
  const [focused, setFocused] = useState(false);

  const onClose = () => {
    setInput('');
    setFocused(false);
    inputRef.current?.blur();
  };

  const inputRef = useRef<HTMLInputElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  useOnClickOutside(containerRef, () => setFocused(false));
  useKeyDown({ key: 'Escape', active: !!input || focused }, onClose);

  return (
    <RelativeContainer ref={containerRef}>
      <Input ref={inputRef} placeholder='Search' icon={SearchIcon} value={input} onChange={(e) => setInput(e.target.value.toLowerCase())} onFocus={() => setFocused(true)} />

      {!!input || focused ? <SearchResults searchText={input} onClose={onClose} /> : null}

      {/* TODO: recent searches...

        {!!input ? (
          <SearchResults
            searchText={input}
            onClose={() => {
              setInput('');
              setFocused(false);
            }}
          />
        ) : focused ? (
          <RecentSearches />
        ) : null} */}
    </RelativeContainer>
  );
};

export { Search };
