import React, { useRef, useState } from 'react';
import { useOnClickOutside } from '@/hooks';
import { RelativeContainer } from '../styled';
import { Input } from '@/reuseable-components';
import { SearchResults } from './search-results';
// import { RecentSearches } from './recent-searches';

const Search = () => {
  const [input, setInput] = useState('');
  const [focused, setFocused] = useState(false);

  const inputRef = useRef<HTMLInputElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  useOnClickOutside(containerRef, () => setFocused(false));

  return (
    <RelativeContainer ref={containerRef}>
      <Input ref={inputRef} placeholder='Search' icon='/icons/common/search.svg' value={input} onChange={(e) => setInput(e.target.value.toLowerCase())} onFocus={() => setFocused(true)} />

      {!!input || focused ? (
        <SearchResults
          searchText={input}
          onClose={() => {
            setInput('');
            setFocused(false);
            inputRef.current?.blur();
          }}
        />
      ) : null}

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
