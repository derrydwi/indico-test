import { Box, Button, TextField } from "@mui/material";
import React, { useState, useEffect } from "react";
import { useAddUser } from "src/hooks/use-add-user";
import { useSnackbar } from "src/hooks/use-snackbar";

const AddUserForm: React.FC = () => {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");

  const addUser = useAddUser();

  const { showSnackbar } = useSnackbar();

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (name.trim() && email.trim()) {
      addUser.mutate(
        { name: name.trim(), email: email.trim() },
        {
          onSuccess: (_data, variables) => {
            showSnackbar(
              `User "${variables.name}" has been added successfully!`,
              "success"
            );
          },
          onError: (_error, variables) => {
            showSnackbar(
              `Failed to add user "${variables.name}". Please try again.`,
              "error"
            );
          },
        }
      );
    }
  };

  // Clear form after successful submission
  useEffect(() => {
    if (addUser.isSuccess) {
      setName("");
      setEmail("");
    }
  }, [addUser.isSuccess]);

  return (
    <Box
      component="form"
      onSubmit={handleSubmit}
      sx={{ display: "flex", gap: 1, mb: 2 }}
    >
      <TextField
        label="Name"
        size="small"
        variant="outlined"
        value={name}
        onChange={(e) => setName(e.target.value)}
        disabled={addUser.isPending}
        required
      />
      <TextField
        label="Email"
        size="small"
        variant="outlined"
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        disabled={addUser.isPending}
        required
      />
      <Button
        type="submit"
        variant="contained"
        color="primary"
        disabled={!name.trim() || !email.trim()}
        loading={addUser.isPending}
      >
        Add User
      </Button>
    </Box>
  );
};

export default AddUserForm;
