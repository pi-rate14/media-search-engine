# Use the official Node.js 14 image as the base image
FROM node:14

# Set the working directory in the container
WORKDIR /app

# Copy the package.json and package-lock.json files to the working directory
COPY package*.json ./

# Install project dependencies
RUN npm install

# Copy the entire application code to the working directory
COPY . .

# Build the React app for production
RUN npm run build

# Expose port 80 for the React app
EXPOSE 3000

# Specify the command to run when the container starts
CMD [ "npm", "run", "start" ]
