import React, { useRef, useState } from 'react';
import { useOnClickOutside } from '@/hooks';
<<<<<<< HEAD
import { RelativeContainer } from './styled';
=======
import { RelativeContainer } from '../styled';
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
import { Input } from '@/reuseable-components';
import { SearchResults } from './search-results';
// import { RecentSearches } from './recent-searches';

const Search = () => {
  const [input, setInput] = useState('');
  const [focused, setFocused] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useOnClickOutside(ref, () => setFocused(false));

  return (
    <RelativeContainer ref={ref}>
<<<<<<< HEAD
      <Input placeholder='Search' icon='/icons/common/search.svg' value={input} onChange={(e) => setInput(e.target.value)} onFocus={() => setFocused(true)} />
=======
      <Input placeholder='Search' icon='/icons/common/search.svg' value={input} onChange={(e) => setInput(e.target.value.toLowerCase())} onFocus={() => setFocused(true)} />
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866

      {!!input || focused ? (
        <SearchResults
          searchText={input}
          onClose={() => {
            setInput('');
            setFocused(false);
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
