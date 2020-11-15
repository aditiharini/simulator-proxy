install.packages("dplyr", repos="http://cran.us.r-project.org")
library(dplyr)
args = commandArgs(trailingOnly=TRUE)

sink(args[2])
data <- read.csv(args[1])
combined_data <- filter(data, type == "combined")
quantile(combined_data$latency, c(.25, .5, .75, .95, .99))
sink()
