import { useMutation } from "@tanstack/react-query";
import axios from "src/lib/axios";
import { queryClient } from "src/lib/react-query";
import type { User } from "src/types/user";

export const useDeleteUser = () => {
  return useMutation({
    mutationKey: ["users/deleteUser"],
    mutationFn: async (id: User["id"]) => {
      const { data } = await axios.delete(`/users/${id}`);
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["users/getUsers"],
      });
    },
  });
};
