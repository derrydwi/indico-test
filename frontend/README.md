# Indico Frontend Assignment

A modern React TypeScript frontend application demonstrating user management with real-time search, CRUD operations, and responsive design using Material-UI and JSONPlaceholder API.

## 🚀 Features

### Core Functionality

- **User Management**: View users with ID, name, email, and company information
- **Add Users**: Create new users with name and email validation
- **Delete Users**: Remove users with confirmation feedback
- **Real-time Search**: Debounced search filtering by user name (300ms delay)
- **Pagination**: Table pagination with configurable rows per page (5, 10, 25)
- **Responsive Design**: Mobile-first approach with Material-UI components
- **Toast Notifications**: Success/error feedback for all user actions
- **Loading States**: Smooth user experience with loading spinners
- **Error Handling**: Graceful error display and retry mechanisms

### Data Features

- **User Fields**: ID, Name, Email, Company Name
- **Client-side Filtering**: Instant search through loaded user data
- **Optimistic UI**: Immediate feedback with rollback on errors
- **API Integration**: JSONPlaceholder for demo user data

### Technical Features

- **Type Safety**: Full TypeScript implementation
- **State Management**: React Query for server state
- **Performance**: Optimized re-renders and data fetching
- **Modern React**: React 19 with hooks and functional components
- **Build Optimization**: Vite for fast development and production builds

## 🏗️ Architecture

```
src/
├── components/          # Reusable UI components
│   ├── add-user-form.tsx    # User creation form
│   ├── loading-spinner.tsx  # Loading state component
│   ├── search-box.tsx       # Search input component
│   └── user-table.tsx       # User list display
├── hooks/               # Custom React hooks
├── lib/                 # Utility functions and configurations
├── providers/           # React context providers
│   ├── react-query-provider.tsx  # React Query setup
│   └── snackbar-provider.tsx     # Notification system
├── types/               # TypeScript type definitions
├── App.tsx              # Main application component
└── main.tsx             # Application entry point
```

## 🛠️ Tech Stack

- **React 19**: Latest React with concurrent features
- **TypeScript**: Full type safety and developer experience
- **Material-UI (MUI)**: Comprehensive component library
- **React Query**: Powerful data fetching and caching
- **Vite**: Next-generation build tool
- **Axios**: HTTP client for API communication
- **ESLint**: Code linting with React and TypeScript rules

## 🚦 Quick Start

### Prerequisites

- Node.js 18+
- npm, yarn, or pnpm

### Installation

1. **Install dependencies**:

   ```bash
   npm install
   # or
   pnpm install
   # or
   yarn install
   ```

2. **Set up environment variables**:

   ```bash
   cp .env.example .env
   ```

3. **Start development server**:

   ```bash
   npm run dev
   ```

4. **Open application**:
   Navigate to http://localhost:5173

### Environment Configuration

Copy `.env.example` to `.env` and configure:

```env
VITE_API_BASE_URL=https://jsonplaceholder.typicode.com
```

> **Note**: This app uses JSONPlaceholder API for demo purposes. The API provides fake user data for testing.

## 🎨 Components

### AddUserForm

- **Purpose**: User creation with basic validation
- **Features**: Name and email input, form reset after success, loading states
- **Validation**: Required fields (name and email must not be empty)
- **Feedback**: Toast notifications for success/error states

### SearchBox

- **Purpose**: Real-time user search with debouncing
- **Features**: 300ms debounced input, placeholder text, full-width design
- **Performance**: Prevents excessive API calls during typing
- **UX**: Immediate visual feedback with search-as-you-type

### UserTable

- **Purpose**: Display paginated user list with actions
- **Features**:
  - **Data Display**: ID, Name, Email, Company columns
  - **Pagination**: 5/10/25 rows per page options
  - **Search Integration**: Filters users by name (client-side)
  - **Delete Action**: Remove users with loading states
  - **Empty States**: Messages for no users or no search results
- **Responsive**: Optimized for mobile and desktop viewing

### LoadingSpinner

- **Purpose**: Consistent loading indicator across components
- **Features**: Centered Material-UI CircularProgress
- **Usage**: Displayed during data fetching operations
- **Usage**: Used across components for loading states

## 🔌 API Integration

### Endpoints Used

The frontend integrates with JSONPlaceholder API:

```typescript
// User operations (JSONPlaceholder endpoints)
GET /users                       # Fetch all users with company info
POST /users                      # Create new user (fake response)
DELETE /users/{id}               # Delete user (fake response)
```

### Data Structure

```typescript
// User type from JSONPlaceholder
type User = {
  id: number;
  name: string;
  email: string;
  company: {
    name: string;
  };
};
```

### React Query Configuration

```typescript
// Query configuration
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // 5 minutes
      refetchOnWindowFocus: false, // Disable refetch on focus
      retry: 3, // Retry failed requests
    },
  },
});
```

## 🧪 Development

### Available Scripts

```bash
# Development
npm run dev          # Start development server
npm run build        # Build for production
npm run preview      # Preview production build

# Code Quality
npm run lint         # Run ESLint
npm run lint:fix     # Fix ESLint issues

# Type Checking
npm run type-check   # Run TypeScript compiler
```

### Project Structure Explained

- **`components/`**: Reusable UI components with single responsibility
- **`hooks/`**: Custom hooks for reusable logic
- **`lib/`**: Utility functions and configurations
- **`providers/`**: React context providers for global state
- **`types/`**: TypeScript interfaces and type definitions

### Code Style

The project uses:

- **ESLint**: Code linting with React and TypeScript rules
- **TypeScript Strict Mode**: Enhanced type checking
- **Functional Components**: Modern React patterns
- **Custom Hooks**: Logic separation and reusability

## 🎯 Features Implementation

### Real-time Search

```typescript
// Debounced search with client-side filtering
const SearchBox = ({ onSearch }) => {
  const [query, setQuery] = useState("");
  const debouncedQuery = useDebounce(query, 300);

  useEffect(() => {
    onSearch(debouncedQuery);
  }, [debouncedQuery, onSearch]);
  // Component filters users by name on the client side
};
```

### User Table with Pagination

```typescript
// Client-side filtering and pagination
const filteredUsers = useMemo(() => {
  if (!searchQuery.trim()) return users;
  return users.filter((user) =>
    user.name.toLowerCase().includes(searchQuery.toLowerCase())
  );
}, [users, searchQuery]);

const paginatedUsers = useMemo(() => {
  const startIndex = page * rowsPerPage;
  return filteredUsers.slice(startIndex, startIndex + rowsPerPage);
}, [filteredUsers, page, rowsPerPage]);
```

### CRUD Operations

```typescript
// Add user with form reset
const addUser = useMutation({
  mutationFn: createUser,
  onSuccess: () => {
    showSnackbar("User created successfully", "success");
    setName(""); // Reset form
    setEmail("");
  },
  onError: () => {
    showSnackbar("Failed to create user", "error");
  },
});

// Delete user with loading state
const deleteUser = useMutation({
  mutationFn: (id: number) => axios.delete(\`/users/\${id}\`),
  onSuccess: () => {
    queryClient.invalidateQueries(['users']);
    showSnackbar("User deleted successfully", "success");
  },
});
```

## 🎨 Styling & Theming

### Material-UI Theme

- **Design System**: Consistent Material Design principles
- **Responsive**: Mobile-first responsive breakpoints
- **Accessibility**: WCAG compliant components
- **Customization**: Theme customization for brand consistency

### Component Styling

```typescript
// Styled components with Material-UI
const StyledContainer = styled(Container)(({ theme }) => ({
  marginTop: theme.spacing(4),
  marginBottom: theme.spacing(4),
}));
```

## 🚀 Performance Optimizations

### React Query Benefits

- **Caching**: Intelligent data caching and invalidation
- **Background Updates**: Fresh data without loading states
- **Optimistic Updates**: Immediate UI feedback
- **Error Recovery**: Automatic retry and error handling

### Vite Optimizations

- **Fast HMR**: Hot module replacement for instant updates
- **Tree Shaking**: Eliminates unused code
- **Code Splitting**: Automatic chunk splitting
- **Asset Optimization**: Optimized images and assets

## 🔧 Build & Deployment

### Production Build

```bash
npm run build
```

### Build Output

- **`dist/`**: Production-ready static files
- **Optimized**: Minified and compressed assets
- **Modern**: ES modules with legacy fallbacks

### Deployment Options

- **Static Hosting**: Netlify, Vercel, GitHub Pages
- **CDN**: Amazon S3 + CloudFront
- **Docker**: Container-based deployment

```dockerfile
# Example Dockerfile
FROM node:18-alpine as builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
```

## 🔒 Security Considerations

### Input Validation

- **Client-side**: Immediate feedback and user experience
- **Server-side**: Backend validation for security
- **Sanitization**: Prevent XSS and injection attacks

### Environment Variables

- **Build-time**: Vite environment variable handling
- **Secure**: No sensitive data in client-side code
- **Configuration**: Environment-specific settings

## 🧪 Testing Strategy

### Recommended Testing Approach

```typescript
// Unit tests for components
import { render, screen } from "@testing-library/react";
import { AddUserForm } from "./add-user-form";

test("validates required fields", async () => {
  render(<AddUserForm />);

  const submitButton = screen.getByRole("button", { name: /submit/i });
  fireEvent.click(submitButton);

  expect(screen.getByText("Name is required")).toBeInTheDocument();
});
```

### Testing Tools (Recommended)

- **Jest**: JavaScript testing framework
- **Testing Library**: React component testing
- **MSW**: API mocking for tests
- **Cypress**: End-to-end testing

## 📱 Responsive Design

### Breakpoint Strategy

- **Mobile First**: Base styles for mobile devices
- **Progressive Enhancement**: Desktop features added progressively
- **Touch-friendly**: Appropriate touch targets and interactions

### Material-UI Breakpoints

```typescript
// Responsive styling
const useStyles = makeStyles((theme) => ({
  container: {
    padding: theme.spacing(2),
    [theme.breakpoints.up("md")]: {
      padding: theme.spacing(4),
    },
  },
}));
```

## 🔮 Future Enhancements

### Potential Improvements

- **User Profiles**: Detailed user management
- **Data Visualization**: Charts and analytics
- **Real-time Updates**: WebSocket integration
- **Offline Support**: Service worker implementation
- **Internationalization**: Multi-language support

### Scalability Considerations

- **State Management**: Consider Zustand or Redux for complex state
- **Code Splitting**: Route-based code splitting
- **Micro-frontends**: Module federation for large teams
- **Performance Monitoring**: Error tracking and analytics

---

This frontend implementation demonstrates modern React development with:

- **Type Safety**: Comprehensive TypeScript usage
- **Performance**: Optimized data fetching and rendering
- **User Experience**: Smooth interactions and feedback
- **Maintainability**: Clean architecture and code organization
- **Scalability**: Foundation for future enhancements

**Status**: ✅ Complete with modern React patterns, TypeScript, and Material-UI integration.
