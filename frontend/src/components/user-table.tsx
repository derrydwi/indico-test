import React, { useMemo, useState } from "react";

import {
  Button,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  TablePagination,
} from "@mui/material";
import { useDeleteUser } from "src/hooks/use-delete-user";
import LoadingSpinner from "src/components/loading-spinner";
import { useGetUsers } from "src/hooks/use-get-users";
import { useSnackbar } from "src/hooks/use-snackbar";

type UserTableProps = {
  searchQuery?: string;
};

const UserTable: React.FC<UserTableProps> = ({ searchQuery = "" }) => {
  const { data: users, isLoading, isError } = useGetUsers();
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(5);

  const deleteUser = useDeleteUser();

  const { showSnackbar } = useSnackbar();

  // Filter users based on search query (client-side filtering)
  const filteredUsers = useMemo(() => {
    if (!users) return [];

    if (!searchQuery.trim()) return users;

    return users.filter((user) =>
      user.name.toLowerCase().includes(searchQuery.toLowerCase())
    );
  }, [users, searchQuery]);

  // Paginate filtered users
  const paginatedUsers = useMemo(() => {
    const startIndex = page * rowsPerPage;
    return filteredUsers.slice(startIndex, startIndex + rowsPerPage);
  }, [filteredUsers, page, rowsPerPage]);

  const handleDelete = (id: number) => {
    deleteUser.mutate(id, {
      onSuccess: (_data, userId) => {
        showSnackbar(
          `User (ID: ${userId}) has been deleted successfully!`,
          "success"
        );
      },
      onError: (_error, userId) => {
        showSnackbar(
          `Failed to delete user (ID: ${userId}). Please try again.`,
          "error"
        );
      },
    });
  };

  const handleChangePage = (_event: unknown, newPage: number) => {
    setPage(newPage);
  };

  const handleChangeRowsPerPage = (
    event: React.ChangeEvent<HTMLInputElement>
  ) => {
    setRowsPerPage(parseInt(event.target.value, 10));
    setPage(0);
  };

  // Reset to first page when search changes
  React.useEffect(() => {
    setPage(0);
  }, [searchQuery]);

  if (isLoading) return <LoadingSpinner />;
  if (isError) return <Typography>Error fetching users</Typography>;

  return (
    <Paper>
      <TableContainer>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>ID</TableCell>
              <TableCell>Name</TableCell>
              <TableCell>Email</TableCell>
              <TableCell>Company</TableCell>
              <TableCell>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {paginatedUsers.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} align="center">
                  <Typography variant="body2" color="textSecondary">
                    {searchQuery.trim()
                      ? `No users found matching "${searchQuery}"`
                      : "No users available"}
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              paginatedUsers.map((user) => (
                <TableRow key={user.id}>
                  <TableCell>{user.id}</TableCell>
                  <TableCell>{user.name}</TableCell>
                  <TableCell>{user.email}</TableCell>
                  <TableCell>{user.company.name}</TableCell>
                  <TableCell>
                    <Button
                      variant="contained"
                      color="secondary"
                      size="small"
                      loading={
                        deleteUser.isPending && deleteUser.variables === user.id
                      }
                      onClick={() => handleDelete(user.id)}
                    >
                      Delete
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>
      <TablePagination
        rowsPerPageOptions={[5, 10, 25]}
        component="div"
        count={filteredUsers.length}
        rowsPerPage={rowsPerPage}
        page={page}
        onPageChange={handleChangePage}
        onRowsPerPageChange={handleChangeRowsPerPage}
        labelRowsPerPage="Users per page:"
      />
    </Paper>
  );
};

export default UserTable;
