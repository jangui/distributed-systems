
# Product Names
CUDA_OBJ = life.o

# Input Names
CUDA_FILES = life.cu
CPP_FILES = 

# ------------------------------------------------------------------------------

# CUDA Compiler and Flags
CUDA_PATH = /usr/local/cuda
CUDA_INC_PATH = $(CUDA_PATH)/include
CUDA_BIN_PATH = $(CUDA_PATH)/bin
CUDA_LIB_PATH = $(CUDA_PATH)/lib64

NVCC = nvcc

# OS-architecture specific flags
# OS-architecture specific flags
ifeq ($(OS_SIZE),32)
NVCC_FLAGS := -m32
else
NVCC_FLAGS := -m64
endif
NVCC_FLAGS += -g -dc -Wno-deprecated-gpu-targets --std=c++11 \
             --expt-relaxed-constexpr
NVCC_INCLUDE =
NVCC_LIBS = -lX11
NVCC_GENCODES = -gencode arch=compute_30,code=sm_30 \
		-gencode arch=compute_35,code=sm_35 \
		-gencode arch=compute_50,code=sm_50 \
		-gencode arch=compute_52,code=sm_52 \
		-gencode arch=compute_60,code=sm_60 \
		-gencode arch=compute_61,code=sm_61 \
		-gencode arch=compute_61,code=compute_61


# ------------------------------------------------------------------------------

# CUDA Linker and Flags
CUDA_LINK_FLAGS = -Wno-deprecated-gpu-targets

# ------------------------------------------------------------------------------

# C++ Compiler and Flags
GPP = g++
FLAGS = -g -Wall -D_REENTRANT -std=c++0x -pthread
INCLUDE = -I$(CUDA_INC_PATH)
LIBS = -L$(CUDA_LIB_PATH) -lcudart -lcufft -lsndfile 
# ------------------------------------------------------------------------------
# Make Rules (Lab 1 specific)
# ------------------------------------------------------------------------------

# C++ Object Files
OBJ_CPU = $(addprefix cpu-, $(notdir $(addsuffix .o, $(CPP_FILES))))
OBJ_CUDA = $(addprefix cuda-, $(notdir $(addsuffix .o, $(CPP_FILES))))

# Top level rules
all: life-cpu life-cuda


life-cpu: life.cu
	$(NVCC) $(CUDA_LINK_FLAGS) $(NVCC_GENCODES) $(NVCC_LIBS) -o $@ $(NVCC_INCLUDE) life.cu

life-cuda: life.cu
	$(NVCC) $(CUDA_LINK_FLAGS) -DCUDA $(NVCC_GENCODES) $(NVCC_LIBS) -o $@ $(NVCC_INCLUDE) life.cu

test-cpu: life-cpu
	srun --x11 --nodes=1 --gres=gpu:1 life-cpu -i spacefiller.lif

test-cuda: life-cuda
	srun --x11 --nodes=1 --gres=gpu:1 life-cuda -i spacefiller.lif

go-cpu: life-cpu 
#	srun --x11 --nodes=1 --gres=gpu:1 life-cpu parabolicsawtooth.lif
	srun --x11 --nodes=1 --gres=gpu:1 life-cpu spacefiller.lif
	#srun --x11 --nodes=1 --gres=gpu:1 life-cuda gosperglidergun.lif

go-cuda: life-cuda 
	#srun --x11 --nodes=1 --gres=gpu:1 life-cuda parabolicsawtooth.lif
	srun --x11 --nodes=1 --gres=gpu:1 life-cuda spacefiller.lif
	#srun --x11 --nodes=1 --gres=gpu:1 life-cuda gosperglidergun.lif

# Clean everything
clean:
	rm -f life-cuda life-cpu *.o *~

.PHONY: clean all
