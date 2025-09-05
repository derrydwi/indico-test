import React, { useState, useEffect } from "react";
import { TextField } from "@mui/material";
import { useDebounce } from "src/hooks/use-debounce";

type SearchBoxProps = {
  onSearch: (query: string) => void;
};

const SearchBox: React.FC<SearchBoxProps> = ({ onSearch }) => {
  const [query, setQuery] = useState("");
  const debouncedQuery = useDebounce(query, 300);

  useEffect(() => {
    onSearch(debouncedQuery);
  }, [debouncedQuery, onSearch]);

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setQuery(event.target.value);
  };

  return (
    <TextField
      label="Search Users by Name"
      size="small"
      variant="outlined"
      value={query}
      onChange={handleChange}
      fullWidth
      margin="normal"
      placeholder="Type to search users..."
      sx={{ mb: 2 }}
    />
  );
};

export default SearchBox;
