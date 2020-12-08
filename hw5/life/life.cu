/*
    Jeff Epstein
    NYU Tandon, CS-UY 3254
    Conway's Life in CUDA
*/

#include <stdio.h>
#include <stdlib.h>
#include <X11/Xlib.h>
#include <unistd.h>
#include <time.h>


/***********************
    Data structures
************************/

#define GRID_SIZE 512
#define CELL_SIZE 2
#define DELAY 10000


struct global {
    char *cells;
    char *cells_next;
    int *cellsInt; // used to make CUDA updates simpler
    // CUDA vars
    int *gpu_cells;
    int *gpu_cells_next;
};

#ifdef CUDA

#define BLOCKS 128
#define BLOCK_THREADS 128

/***********************
    Game of Life, GPU version
************************/

static void HandleError( cudaError_t err,
                         const char *file,
                         int line ) {
    if (err != cudaSuccess) {
        printf( "%s in %s at line %d\n", cudaGetErrorString( err ),
                file, line );
        exit( EXIT_FAILURE );
    }
}
#define HANDLE_ERROR( err ) (HandleError( err, __FILE__, __LINE__ ))

__device__ int count_neighbors(int *gpu_cells, uint tid, int grid_size) {
    int x = ((tid / grid_size)+grid_size/2)%grid_size;
    int y = ((tid % grid_size)+grid_size/2)%grid_size;
    int count = 0;
    for (int i=x-1; i<=x+1; i++)
        for (int j=y-1; j<=y+1; j++)
            if (i!=x || j!=y)
                count += gpu_cells[((i+grid_size/2)%grid_size) * grid_size + ((j+grid_size/2)%grid_size)]; 
    return count;
}


__device__ void update_cell(int *gpu_cells, int *gpu_cells_next, uint tid, int grid_size) {
    int neighbors = count_neighbors(gpu_cells, tid, grid_size);
    gpu_cells_next[tid] = neighbors==3 || (gpu_cells[tid] && (neighbors == 2 || neighbors == 3));
}

__global__ void kernel(int *gpu_cells, int *gpu_cells_next, int grid_size) {
    // get thread index
    uint thread_index = threadIdx.x + blockDim.x * blockIdx.x;

    while (thread_index < grid_size*grid_size) {
        // Do computation for this thread index
        update_cell(gpu_cells, gpu_cells_next, thread_index, grid_size);
        // update thread index
        thread_index += blockDim.x * gridDim.x;
    }
}

// copy gpu_cells_next to gpu_cells
// must be run after full update finished
__global__ void copyGPUcells(int *gpu_cells, int *gpu_cells_next, int grid_size) {
    // get thread index
    uint thread_index = threadIdx.x + blockDim.x * blockIdx.x;
    while (thread_index < grid_size*grid_size) {
        gpu_cells[thread_index] = gpu_cells_next[thread_index];
        thread_index += blockDim.x * gridDim.x;
    }
}

void init_global(struct global *g) {
    // Initialize the global data structure
    const int size = GRID_SIZE*GRID_SIZE/8;
    g->cells = (char*)calloc(size, sizeof(char)); // calloc will init mem to 0's
    g->cellsInt = (int*)calloc(GRID_SIZE*GRID_SIZE, sizeof(int));

    if (g->cells==NULL || g->cellsInt==NULL) {
        fprintf(stderr, "Error: alloc failed\n");
        exit(1);
    }

    // allocate space on gpu for cells
    HANDLE_ERROR(cudaMalloc((void**)&(g->gpu_cells), GRID_SIZE*GRID_SIZE * sizeof(int)));
    HANDLE_ERROR(cudaMalloc((void**)&(g->gpu_cells_next), GRID_SIZE*GRID_SIZE * sizeof(int)));

    // set initial state to all 0's
    HANDLE_ERROR(cudaMemcpy(g->gpu_cells, g->cellsInt, GRID_SIZE*GRID_SIZE * sizeof(int), cudaMemcpyHostToDevice));
}

bool get_cell(struct global *g, int x, int y) {
    return (g->cells[(y*GRID_SIZE + x)/8] & (1<<(x%8))) != 0;
}

void set_cell(struct global *g, int x, int y, bool val) {
    if (val)
        g->cells[(y*GRID_SIZE+x)/8] |= (1<<(x%8));
    else
        g->cells[(y*GRID_SIZE+x)/8] &= ~(1<<(x%8));
}

void update(struct global *global) {
    // Conway's life algorithm on the GPU

    // update cells
    kernel<<<BLOCKS, BLOCK_THREADS>>>(global->gpu_cells, global->gpu_cells_next, GRID_SIZE);

    // copy gpu_cells_next to gpu_cells
    copyGPUcells<<<BLOCKS, BLOCK_THREADS>>>(global->gpu_cells, global->gpu_cells_next, GRID_SIZE);

    // check if errors
    cudaError err = cudaGetLastError();
    if (cudaSuccess != err)
        fprintf(stderr, "Error %s\n", cudaGetErrorString(err));

    // copy data from gpu to cpu
    cudaMemcpy(global->cellsInt, global->gpu_cells, GRID_SIZE*GRID_SIZE * sizeof(int), cudaMemcpyDeviceToHost);

    // set cpu cells (char) appropriately from cellsInt
    int lower = -GRID_SIZE/2 + 1;
    int upper = GRID_SIZE/2 - 1;
    for (int x = lower; x < upper; x++) {
        for (int y = lower; y < upper; y++) {
            int index = ((x+GRID_SIZE/2)%GRID_SIZE)*GRID_SIZE+((y+GRID_SIZE/2)%GRID_SIZE);
            set_cell(global,x+GRID_SIZE/2,y+GRID_SIZE/2,global->cellsInt[index]);
        }
    }
}

#else

/***********************
    Game of Life, CPU version
************************/

/*
    Allocate memory for data structures
    and initialize data
*/
void init_global(struct global *g) {
    const int size = GRID_SIZE*GRID_SIZE/8;
    g->cells=(char*)malloc(size);
    g->cells_next=(char*)malloc(size);
    if (g->cells==NULL || g->cells_next==NULL) {
        fprintf(stderr, "Error: can't alloc data\n");
        exit(1);
    }
    for (int i=0; i<size; i++)
        g->cells[i]=0;
}

/*
    Returns true if a cell is alive at the given location
*/
bool get_cell(struct global *g, int x, int y) {
    return (g->cells[(y*GRID_SIZE + x)/8] & (1<<(x%8))) != 0;
}

void set_cell_next(struct global *g, int x, int y, bool val) {
    if (val)
        g->cells_next[(y*GRID_SIZE+x)/8] |= (1<<(x%8));
    else
        g->cells_next[(y*GRID_SIZE+x)/8] &= ~(1<<(x%8));
}

/*
    Set a cell alive or dead at the given location
*/
void set_cell(struct global *g, int x, int y, bool val) {
    if (val)
        g->cells[(y*GRID_SIZE+x)/8] |= (1<<(x%8));
    else
        g->cells[(y*GRID_SIZE+x)/8] &= ~(1<<(x%8));
}

/*
    Count neighbors of given cell
*/
int count_neighbors(struct global *g, int x, int y) {
    int count =0;
    for (int i=x-1; i<=x+1; i++)
        for (int j=y-1; j<=y+1; j++)
            if (i!=x || j!=y)
                count += get_cell(g,i,j); 
    return count;
}

/*
    Perform a complete step, storing the new state
    in global->cells
*/
void update(struct global *global) {
    for (int x=1; x<GRID_SIZE-1; x++)
        for (int y=1; y<GRID_SIZE-1; y++) {
            int neighbors = count_neighbors(global, x, y);
            bool newstate = 
                neighbors==3 || (get_cell(global,x,y) && (neighbors == 2 || neighbors == 3));
            set_cell_next(global,x,y,newstate);
        }    
    char *temp=global->cells;
    global->cells = global->cells_next;
    global->cells_next = temp;
}

#endif

/***********************
    X Window stuff
************************/

#define COLOR_RED "#FF0000"
#define COLOR_GREEN "#00FF00"
#define COLOR_BLACK "#000000"
#define COLOR_WHITE "#FFFFFF"

struct display
{
    Display         *display;
    Window          window;
    int             screen;
    Atom            delete_window;
    GC              gc;
    XColor          color1;
    XColor          color2;
    Colormap        colormap;
};

void init_display(struct display *dpy) {
        dpy->display = XOpenDisplay(NULL);
        if(dpy->display == NULL)
        {
            fprintf(stderr, "Error: could not open X dpy->display\n");
            exit(1);
        }
        dpy->screen = DefaultScreen(dpy->display);
        dpy->window = XCreateSimpleWindow(dpy->display, RootWindow(dpy->display, dpy->screen),
                0, 0, GRID_SIZE * CELL_SIZE, 
                GRID_SIZE * CELL_SIZE, 1,
                BlackPixel(dpy->display, dpy->screen), WhitePixel(dpy->display, dpy->screen));
        dpy->delete_window = XInternAtom(dpy->display, "WM_DELETE_WINDOW", 0);
        XSetWMProtocols(dpy->display, dpy->window, &dpy->delete_window, 1);
        XSelectInput(dpy->display, dpy->window, ExposureMask | KeyPressMask);
        XMapWindow(dpy->display, dpy->window);
        dpy->colormap = DefaultColormap(dpy->display, 0);
        dpy->gc = XCreateGC(dpy->display, dpy->window, 0, 0);
        XParseColor(dpy->display, dpy->colormap, COLOR_BLACK, &dpy->color1);
        XParseColor(dpy->display, dpy->colormap, COLOR_WHITE, &dpy->color2);
        XAllocColor(dpy->display, dpy->colormap, &dpy->color1);
        XAllocColor(dpy->display, dpy->colormap, &dpy->color2);

        XSelectInput(dpy->display,dpy->window, 
            KeyPressMask | KeyReleaseMask | ButtonPressMask | ButtonReleaseMask);

}

bool lookup_cell(struct global *g, int x, int y) {
    return (g->cells[(y*GRID_SIZE + x)/8] & (1<<(x%8))) != 0;
}


void do_display(struct global *global, struct display *dpy)
{
    XSetBackground(dpy->display, dpy->gc, dpy->color2.pixel);
    XClearWindow(dpy->display, dpy->window);

    for (int x=0; x<GRID_SIZE; x++)
        for (int y=0; y<GRID_SIZE; y++)
        {
            bool state = get_cell(global, x, y);
            if (state) {
                XSetForeground(dpy->display, dpy->gc, dpy->color1.pixel);
                XFillRectangle(dpy->display, dpy->window, dpy->gc, x*CELL_SIZE, y*CELL_SIZE, CELL_SIZE, CELL_SIZE);        
//              XDrawPoint(dpy->display, dpy->window, dpy->gc, x*CELL_SIZE, y*CELL_SIZE);
            }
        }

    XFlush(dpy->display);
}

void close_display(struct display *dpy)
{
    XDestroyWindow(dpy->display, dpy->window);
    XCloseDisplay(dpy->display);
}

/***********************
    Main program
************************/

void load_life(struct global *g, const char *fname) {
    char *line=NULL;
    size_t len = 0;
    ssize_t nread;
    int x,y;
    FILE *f = fopen(fname, "r");
    if (f==NULL) {
        fprintf(stderr,"Can't open file\n");
        exit(1);
    }
    while ((nread = getline(&line, &len, f)) != -1) {
        if (line[0]=='#')
            continue;
        if (nread<=1)
            continue;
        if (line[0]==13 || line[0]==10)
            continue;
        if (sscanf(line, "%d %d", &x, &y) != 2)
            continue;
        set_cell(g,x+GRID_SIZE/2,y+GRID_SIZE/2,1);
#ifdef CUDA
        // we only need to copy data to over to GPU once
        int index = ((x+GRID_SIZE/2)%GRID_SIZE)*GRID_SIZE+((y+GRID_SIZE/2)%GRID_SIZE);
        g->cellsInt[index] = 1;
        cudaMemcpy(g->gpu_cells, g->cellsInt, GRID_SIZE*GRID_SIZE * sizeof(int), cudaMemcpyHostToDevice);
#endif
    }
    if (line)
        free(line);
    fclose(f);
}

void do_life(struct global *global) {
    bool running=1;
    struct display dpy;
    init_display(&dpy);
    while (running) {
        do_display(global, &dpy);
        usleep(DELAY);
        update(global);

        if (XPending(dpy.display)) {
            XEvent event;
            XNextEvent(dpy.display, &event);
            switch (event.type)
            {
                case ClientMessage:
                    if (event.xclient.data.l[0] == dpy.delete_window)
                        running=0;
                    break;
                case KeyPress:
                case ButtonPress:
                    running=0;
                    break;
                default:
                    break;
            }
        }
    }
    close_display(&dpy);
}

void perf_test(struct global *global) {
    int counter=10000;
    clock_t start = clock();
    clock_t diff;
    int msec;

    printf("Running performance test with %d iterations...\n", counter);
    fflush(stdout);

    while (counter>0) {
        update(global);
        counter--;
    }
    diff = clock() - start;
    msec = diff * 1000 / CLOCKS_PER_SEC;
    printf("Time taken %d seconds %d milliseconds\n", msec/1000, msec%1000);
}

int main(int argc, char *argv[]) {
    bool gui=1;
    struct global global;
    init_global(&global);

    #ifdef CUDA
        printf("Starting CUDA version of life....\n");
    #else
        printf("Starting CPU version of life....\n");
    #endif

    int argi;
    for (argi = 1; argi<argc; argi++)
        if (argv[argi][0]=='-' && argv[argi][1]=='i' && argv[argi][2]=='\0')
            gui=0;
        else
            break;

    if (argi==argc-1)
        load_life(&global, argv[argi]);
    else {
        fprintf(stderr,"Syntax: %s [-i] fname.lif\n", argv[0]);
        exit(1);
    }

    if (gui)
        do_life(&global);
    else
        perf_test(&global);
        
    return 0;
}
