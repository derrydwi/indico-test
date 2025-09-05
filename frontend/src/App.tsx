import { useState } from "react";
import { Container, Typography } from "@mui/material";
import { ReactQueryProvider } from "src/providers/react-query-provider";
import { SnackbarProvider } from "src/providers/snackbar-provider";
import SearchBox from "src/components/search-box";
import AddUserForm from "src/components/add-user-form";
import UserTable from "src/components/user-table";

const App = () => {
  const [searchQuery, setSearchQuery] = useState("");

  const handleSearch = (query: string) => {
    setSearchQuery(query);
  };

  return (
    <ReactQueryProvider>
      <SnackbarProvider>
        <Container maxWidth="lg" sx={{ mx: "auto", my: 4 }}>
          <Typography variant="h4" gutterBottom>
            User Management
          </Typography>
          <SearchBox onSearch={handleSearch} />
          <AddUserForm />
          <UserTable searchQuery={searchQuery} />
        </Container>
      </SnackbarProvider>
    </ReactQueryProvider>
  );
};

export default App;
