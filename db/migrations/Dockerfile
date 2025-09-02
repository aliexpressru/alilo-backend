FROM gomicro/goose

# Copy SQLs and script into image
ADD ./*.sql /migrations/
ADD entrypoint.sh /migrations/

# Make entrypoint script executable
# RUN chmod +x /migrations/entrypoint.sh

# Set entrypoint 

ENTRYPOINT ["/migrations/entrypoint.sh"]

# CMD ["sh", "-c", "/migrations/entrypoint.sh"]
