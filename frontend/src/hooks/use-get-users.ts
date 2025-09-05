import { useQuery } from "@tanstack/react-query";
import axios from "src/lib/axios";
import type { User } from "src/types/user";

export const useGetUsers = () => {
  return useQuery({
    queryKey: ["users/getUsers"],
    queryFn: async () => {
      const { data } = await axios.get<User[]>("/users");
      return data;
    },
  });
};
