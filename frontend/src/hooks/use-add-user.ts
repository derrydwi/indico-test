import { useMutation } from "@tanstack/react-query";
import axios from "src/lib/axios";
import { queryClient } from "src/lib/react-query";
import type { User } from "src/types/user";

export const useAddUser = () => {
  return useMutation({
    mutationKey: ["users/addUser"],
    mutationFn: async (newUser: Pick<User, "name" | "email">) => {
      const { data } = await axios.post("/users", newUser);
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["users/getUsers"],
      });
    },
  });
};
